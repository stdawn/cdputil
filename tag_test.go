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
	"strings"
	"testing"
	"time"
)

func TestNewTag(t *testing.T) {
	GetBrowserInfo().WithName("msedge.exe").
		WithUrl("C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe").
		WithPort(9223).
		WithWindowWidth(1000).
		WithWindowHeight(500)

	tag, err := NewTag(30 * time.Minute)
	if err != nil {
		fmt.Println(err)
		return
	}
	tag.RequestTaskValidTypesMap[network.ResourceTypeXHR] = true
	tag.IsWaitCurrentRequestTasksFinished = true

	urlStr := ""

	err = tag.RunMain(
		chromedp.Navigate("about:blank"))
	if err != nil {
		fmt.Println(err)
		return
	}

	js := ""
	requestId := ""
	tag.RequestPausedCallback = func(rp *fetch.EventRequestPaused) *fetch.ContinueRequestParams {
		if len(requestId) < 1 && strings.HasPrefix(rp.Request.URL, urlStr) {
			requestId = string(rp.NetworkID)
			//doSomething
		}
		headers := rp.Request.Headers
		hs := make([]*fetch.HeaderEntry, 0)
		for k, v := range headers {
			if strings.HasPrefix(k, "sec") {
				continue
			}
			hs = append(hs, &fetch.HeaderEntry{Name: k, Value: v.(string)})
		}
		return fetch.ContinueRequest(rp.RequestID).WithHeaders(hs)

	}

	defer tag.Cancel()
	err = tag.RunMain(
		chromedp.Tasks{
			network.Enable(),
			chromedp.Evaluate(js, nil),
			chromedp.Navigate(urlStr),
			chromedp.Sleep(time.Hour),
		}...,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	rt := tag.GetRequestTask(requestId)
	if rt != nil {
		fmt.Println(fmt.Sprintf("respones=%s", rt.ResponseBody))
	}
}
