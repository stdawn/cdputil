/**
 * @Time: 2023/11/27 16:38
 * @Author: LiuKun
 * @File: tag_test.go
 * @Description:
 */

package cdputil

import (
	"fmt"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

var success []int
var success1 []int

func TestTag(t *testing.T) {

	GetBrowserInfo().WithName("msedge.exe").
		WithUrl("C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe").
		WithPort(9223).
		WithWindowWidth(1000).
		WithWindowHeight(500)

	wg := new(sync.WaitGroup)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		index := i
		go func() {
			defer wg.Done()
			request(index + 1)
		}()
	}
	wg.Wait()

	fmt.Println("success0 count=", len(success), ": ", success)
	fmt.Println("success1:count=", len(success1), ": ", success1)

}

func request(index int) {
	tag, err := NewTag(30 * time.Minute)
	if err != nil {
		fmt.Println(err)
		return
	}
	tag.RequestTaskValidTypesMap[network.ResourceTypeXHR] = true
	tag.IsWaitCurrentRequestTasksFinished = false

	urlstr := "https://www.iwencai.com/customized/chart/get-robot-data"
	urlstr1 := "https://www.iwencai.com/gateway/urp/v7/landing/getDataList"

	requestId := ""
	requestId1 := ""

	tag.RequestPausedCallback = func(rp *fetch.EventRequestPaused) *fetch.ContinueRequestParams {
		if len(requestId) < 1 && rp.Request.URL == urlstr {
			requestId = string(rp.NetworkID)
			postData := rp.Request.PostData
			reg := regexp.MustCompile("\"perpage\":\\w+,")
			str := reg.FindString(postData)
			if len(str) > 0 {
				postData = strings.ReplaceAll(postData, str, "\"perpage\":100,")
			}
			return fetch.ContinueRequest(rp.RequestID).WithPostData(postData)
		} else if len(requestId1) < 1 && rp.Request.URL == urlstr1 {
			requestId1 = string(rp.NetworkID)
			return nil
		}
		return nil
	}
	defer tag.Cancel()
	err = tag.RunMain(
		chromedp.Navigate("https://www.iwencai.com/unifiedwap/result?w=%E6%B6%A8%E8%B7%8C%E5%B9%85&querytype=stock"),
		tag.WaitRequestTaskFinish(&requestId),
		//点击事件只适用于单线程
		//chromedp.Click("#iwcTableWrapper > div.xuangu-bottom-tool > div.pcwencai-pagination-wrap > div.pager > ul > li:nth-child(3) > a"),
		//tag.WaitRequestTaskFinish(&requestId1),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	rt := tag.GetRequestTask(requestId)
	if rt != nil {

		success = append(success, index)
		fmt.Println(fmt.Sprintf("url0: 第%d个收到%dBytes", index, len(rt.ResponseBody)))
	}

	rt1 := tag.GetRequestTask(requestId1)
	if rt1 != nil {

		success1 = append(success1, index)
		fmt.Println(fmt.Sprintf("url1: 第%d个收到%dBytes", index, len(rt1.ResponseBody)))
	}
}
