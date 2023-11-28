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
	ctx     context.Context
	cancel  context.CancelFunc
	cancel1 context.CancelFunc

	RequestPausedCallback func(rp *fetch.EventRequestPaused) *fetch.ContinueRequestParams

	RequestMap sync.Map //map[string]*RequestTask
}

// NewTag 创建一个Tag
func NewTag(timeout time.Duration) (*Tag, error) {
	rCtx, _, err := getRemoteContext()
	if err != nil {
		return nil, err
	}
	tag := new(Tag)

	tag.ctx, tag.cancel = chromedp.NewContext(rCtx)
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
							rTask := tag.getRequestTask(ev.NetworkID, false)
							rTask.ErrorText = fmt.Sprintf("fetch continue request error:%s", err.Error())
							rTask.IsFinished = true
						}
					}()
				} else {
					go func() {
						rTask := tag.getRequestTask(ev.NetworkID, false)
						rTask.HasRewrite = true
						// 重写参数(复制continueRequest)
						rTask.RewriteParams = util.DeepCopy(continueRequest).(*fetch.ContinueRequestParams)

						if len(continueRequest.PostData) > 0 {
							continueRequest.PostData = base64.StdEncoding.EncodeToString([]byte(continueRequest.PostData))
						}

						err = tag.Run(continueRequest)
						if err != nil {
							rTask.ErrorText = fmt.Sprintf("fetch continue request error:%s", err.Error())
							rTask.IsFinished = true
						}
					}()
				}
			}

		case *network.EventRequestWillBeSent:
			rTask := tag.getRequestTask(ev.RequestID, false)
			rTask.Request = ev.Request
			rTask.Type = ev.Type
			rTask.DocumentUrl = ev.DocumentURL

		case *network.EventResponseReceived:
			rTask := tag.getRequestTask(ev.RequestID, true)
			if rTask != nil {
				rTask.Response = ev.Response
			}

		case *network.EventLoadingFailed:
			rTask := tag.getRequestTask(ev.RequestID, true)
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
			rTask := tag.getRequestTask(ev.RequestID, true)
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
	as = append(as, t.checkRequestTaskIsFinished())
	err := t.Run(as...)
	if err != nil {
		//ws控制连接失败时
		if strings.Contains(err.Error(), "failed to modify wsURL") {
			remoteContext = nil
			remoteCancel = nil
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
	t.RequestMap.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(*RequestTask))
	})
}

// 获取请求任务
func (t *Tag) getRequestTask(requestId network.RequestID, canNil bool) *RequestTask {
	id := string(requestId)
	rt, ok := t.RequestMap.Load(id)
	if ok {
		return rt.(*RequestTask)
	}
	if canNil {
		return nil
	}
	rTask := new(RequestTask)
	rTask.RequestId = id
	t.RequestMap.Store(id, rTask)
	return rTask
}

func (t *Tag) checkRequestTaskIsFinished() chromedp.ActionFunc {
	return func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				isFinished := true
				t.RequestMap.Range(func(key, value interface{}) bool {
					if !value.(*RequestTask).IsFinished {
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
