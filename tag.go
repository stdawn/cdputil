/**
 * @Time: 2023/11/24 14:31
 * @Author: LiuKun
 * @File: tag.go
 * @Description:
 */

package cdputil

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/stdawn/util"
	"strings"
	"sync"
	"time"
)

// Tag 浏览器标签
type Tag struct {
	ctx            context.Context    //timeout context
	cancel         context.CancelFunc //context cancel
	cancel1        context.CancelFunc //timeout context cancel
	rCtx           context.Context    //公用远程context
	requestTaskMap sync.Map           //map[string]*RequestTask

	IsWaitCurrentRequestTasksFinished bool //是否等待当前所有的请求任务完成, 默认为true
	RequestPausedCallback             func(rp *fetch.EventRequestPaused) *fetch.ContinueRequestParams
	RequestTaskValidTypesMap          map[network.ResourceType]bool // 需要保存的请求任务类型，如果为空，则全部保存

}

// NewTag 创建一个Tag
func NewTag(timeout time.Duration) (*Tag, error) {
	rCtx, _, err := getRemoteContext()
	if err != nil {
		return nil, err
	}
	tag := new(Tag)
	tag.rCtx = rCtx

	tag.IsWaitCurrentRequestTasksFinished = true
	tag.RequestTaskValidTypesMap = make(map[network.ResourceType]bool)
	tag.ctx, tag.cancel = chromedp.NewContext(tag.rCtx)
	tag.ctx, tag.cancel1 = context.WithTimeout(tag.ctx, timeout)

	chromedp.ListenTarget(tag.ctx, func(v interface{}) {
		switch ev := v.(type) {
		case *fetch.EventRequestPaused:
			if tag.RequestPausedCallback != nil {
				continueRequest := tag.RequestPausedCallback(ev)
				if continueRequest == nil {
					go func() {
						err = tag.Run(fetch.ContinueRequest(ev.RequestID))
						if err != nil {
							if tag.isResourceTypeValid(ev.ResourceType) {
								rTask := tag.getRequestTaskWithoutNil(ev.NetworkID)
								rTask.ErrorText = fmt.Sprintf("fetch continue request error:%s", err.Error())
								rTask.IsFinished = true
							}
						}
					}()
				} else {
					go func() {
						if tag.isResourceTypeValid(ev.ResourceType) {
							rTask := tag.getRequestTaskWithoutNil(ev.NetworkID)
							rTask.HasRewrite = true
							// 重写参数(复制continueRequest)
							rTask.RewriteParams = util.DeepCopy(continueRequest).(*fetch.ContinueRequestParams)
						}

						if len(continueRequest.PostData) > 0 {
							continueRequest.PostData = base64.StdEncoding.EncodeToString([]byte(continueRequest.PostData))
						}

						err = tag.Run(continueRequest)
						if err != nil {
							rTask := tag.GetRequestTask(string(ev.NetworkID))
							if rTask != nil {
								rTask.ErrorText = fmt.Sprintf("fetch continue request error:%s", err.Error())
								rTask.IsFinished = true
							}
						}
					}()
				}
			}

		case *network.EventRequestWillBeSent:
			if tag.isResourceTypeValid(ev.Type) {
				rTask := tag.getRequestTaskWithoutNil(ev.RequestID)
				rTask.Request = ev.Request
				rTask.Type = ev.Type
				rTask.DocumentUrl = ev.DocumentURL
			}

		case *network.EventResponseReceived:
			rTask := tag.GetRequestTask(string(ev.RequestID))
			if rTask != nil {
				rTask.Response = ev.Response
			}

		case *network.EventLoadingFailed:
			rTask := tag.GetRequestTask(string(ev.RequestID))
			if rTask != nil {
				if len(ev.ErrorText) > 0 {
					rTask.ErrorText += ev.ErrorText
				}
				if len(ev.BlockedReason.String()) > 0 {
					rTask.ErrorText += "\n"
					rTask.ErrorText += ev.BlockedReason.String()
				}
				if len(rTask.ErrorText) < 1 {
					rTask.ErrorText = "unknown error"
				}
			}

		case *network.EventLoadingFinished:
			rTask := tag.GetRequestTask(string(ev.RequestID))
			if rTask != nil {
				if rTask.Success() {
					go func() {
						_ = tag.Run(chromedp.ActionFunc(func(ctx context.Context) error {
							//保存响应内容
							buf, e := network.GetResponseBody(ev.RequestID).Do(ctx)
							rTask.ResponseBody = string(buf)
							rTask.IsFinished = true
							return e
						}))

					}()
				} else {
					rTask.IsFinished = true
				}
			}
		}
	})
	return tag, nil
}

func (t *Tag) Run(actions ...chromedp.Action) error {
	return chromedp.Run(t.ctx, actions...)
}

func (t *Tag) RunMain(actions ...chromedp.Action) error {

	as := make([]chromedp.Action, 0)
	if t.RequestPausedCallback != nil {
		as = append(as, fetch.Enable())
	}
	as = append(as, actions...)
	if t.IsWaitCurrentRequestTasksFinished {
		as = append(as, t.checkRequestTasksIsFinished())
	}
	err := t.Run(as...)
	if err != nil {
		//ws控制连接失败时
		if strings.Contains(err.Error(), fmt.Sprintf("could not dial \"ws://localhost:%d", GetBrowserInfo().port)) {
			clearRemoteContext(t.rCtx)
		}
		return err
	}
	return nil

}

// Cancel 关闭Tag
func (t *Tag) Cancel() {
	t.cancel1()
	t.cancel()
}

// RangeRequestTask 遍历请求任务
func (t *Tag) RangeRequestTask(f func(key string, rt *RequestTask) bool) {
	t.requestTaskMap.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(*RequestTask))
	})
}

// GetRequestTask 获取请求任务,可为nil
func (t *Tag) GetRequestTask(requestId string) *RequestTask {
	r, ok := t.requestTaskMap.Load(requestId)
	if ok {
		return r.(*RequestTask)
	}
	return nil
}

// WaitRequestTaskFinish 等待请求任务完成requestIdPts为requestId的指针地址
func (t *Tag) WaitRequestTaskFinish(requestIdPts ...*string) chromedp.ActionFunc {

	return func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				isFinished := true
				for _, pt := range requestIdPts {
					requestId := *pt
					if len(requestId) < 1 {
						isFinished = false
						break
					}
					rTask := t.GetRequestTask(requestId)
					if rTask == nil {
						isFinished = false
						break
					}
					if !rTask.IsFinished {
						isFinished = false
						break
					}
				}
				if isFinished {
					return nil
				}
				time.Sleep(time.Millisecond * 100)
			}
		}
	}

}

// 获取请求任务
func (t *Tag) getRequestTaskWithoutNil(requestId network.RequestID) *RequestTask {
	id := string(requestId)

	rTask := t.GetRequestTask(id)
	if rTask != nil {
		return rTask
	}
	rTask = new(RequestTask)
	rTask.RequestId = id
	t.requestTaskMap.Store(id, rTask)
	return rTask
}

// 检查资源类型是否有效
func (t *Tag) isResourceTypeValid(rType network.ResourceType) bool {
	if len(t.RequestTaskValidTypesMap) < 1 {
		return true
	}
	b, ok := t.RequestTaskValidTypesMap[rType]
	if ok {
		return b
	}
	return false
}

// 检测所有监控的请求任务是否完成
func (t *Tag) checkRequestTasksIsFinished() chromedp.ActionFunc {
	return func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				isFinished := true
				t.RangeRequestTask(func(key string, rt *RequestTask) bool {
					if !rt.IsFinished {
						isFinished = false
						return false
					}
					return true
				})

				if isFinished {
					return nil
				}
				time.Sleep(time.Millisecond * 100)
			}
		}
	}
}
