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
	"github.com/chromedp/chromedp"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestTag(t *testing.T) {

	tag, err := NewTag(10 * time.Minute)
	if err != nil {
		fmt.Println(err)
		return
	}

	var urlstr = "https://www.iwencai.com/customized/chart/get-robot-data"

	tag.RequestPausedCallback = func(rp *fetch.EventRequestPaused) *fetch.ContinueRequestParams {
		if rp.Request.URL == urlstr {
			postData := rp.Request.PostData
			reg := regexp.MustCompile("\"perpage\":\\w+,")
			str := reg.FindString(postData)
			if len(str) > 0 {
				postData = strings.ReplaceAll(postData, str, "\"perpage\":100,")
			}
			return fetch.ContinueRequest(rp.RequestID).WithPostData(postData)
		}
		return nil
	}
	defer tag.Cancel()
	err = tag.RunMain(
		chromedp.Navigate("https://www.iwencai.com/unifiedwap/result?w=%E6%B6%A8%E8%B7%8C%E5%B9%85&querytype=stock"),
		//chromedp.WaitVisible("#iwcTableWrapper > div.xuangu-bottom-tool > div.pcwencai-pagination-wrap > div.drop-down-box"),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	tag.RequestMap.Range(func(key, value interface{}) bool {
		if value.(*RequestTask).Request.URL == urlstr {
			fmt.Println(value.(*RequestTask).ResponseBody)
		}
		return true
	})

}
