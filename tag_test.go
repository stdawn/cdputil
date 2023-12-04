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

	urlstr := "https://www.baidu.com/sugrec"

	requestId := ""
	tag.RequestPausedCallback = func(rp *fetch.EventRequestPaused) *fetch.ContinueRequestParams {
		if len(requestId) < 1 && strings.HasPrefix(rp.Request.URL, urlstr) {
			requestId = string(rp.NetworkID)
			return nil
		}
		return nil
	}
	defer tag.Cancel()
	err = tag.RunMain(
		chromedp.Navigate("https://www.baidu.com"),
		tag.WaitRequestTaskFinish(&requestId),
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
