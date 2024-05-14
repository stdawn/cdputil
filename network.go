/**
 * @Time: 2024/5/14 10:26
 * @Author: LiuKun
 * @File: network.go
 * @Description:
 */

package cdputil

import (
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"time"
)

type RequestMethod string

const (
	RequestMethodGet     RequestMethod = "GET"
	RequestMethodPost    RequestMethod = "POST"
	RequestMethodPut     RequestMethod = "PUT"
	RequestMethodDelete  RequestMethod = "DELETE"
	RequestMethodHead    RequestMethod = "HEAD"
	RequestMethodOptions RequestMethod = "OPTIONS"
	RequestMethodTrace   RequestMethod = "TRACE"
	RequestMethodConnect RequestMethod = "CONNECT"
)

// HasBody 是否有body
func (rm RequestMethod) HasBody() bool {
	return rm != RequestMethodGet && rm != RequestMethodHead
}

// RequestRetry 带重试请求
func RequestRetry(retryCount int, method RequestMethod, urlStr, params string, headers map[string]string, tag *Tag, baseUrl string) (string, error) {
	var response = ""
	var err error = nil
	for i := 0; i < retryCount; i++ {
		response, err = Request(method, urlStr, params, headers, tag, baseUrl)
		if err == nil {
			return response, nil
		}
	}
	return "", err
}

// Request 请求
func Request(method RequestMethod, urlStr, params string, headers map[string]string, tag *Tag, baseUrl string) (string, error) {

	useNavigate := false
	if !method.HasBody() {
		urlStr = urlStr + "?" + params
		if len(baseUrl) < 1 {
			//采用Navigation
			useNavigate = true
		}
	}
	var err error = nil
	if tag == nil {
		tag, err = NewTag(10 * time.Minute)
		if err != nil {
			return "", err
		}
		tag.IsWaitCurrentRequestTasksFinished = false
		defer tag.Cancel()
	}
	requestId := ""
	tag.RequestPausedCallback = func(rp *fetch.EventRequestPaused) *fetch.ContinueRequestParams {
		if len(requestId) < 1 && rp.Request.URL == urlStr {
			requestId = string(rp.NetworkID)
			return nil
		}
		return nil
	}

	if useNavigate {
		ts := make(chromedp.Tasks, 0)
		if len(headers) > 0 {
			ts = append(ts, network.SetExtraHTTPHeaders(HeaderFromMap(headers)))
		}
		ts = append(ts,
			chromedp.Navigate(urlStr),
			tag.WaitRequestTaskFinish(&requestId),
		)
		err = tag.RunMain(ts...)
		if err != nil {
			return "", err
		}
	} else {

		cUrl := ""
		err = tag.RunMain(
			chromedp.Evaluate("window.location.href", &cUrl),
		)
		if err != nil {
			return "", err
		}
		ts := make(chromedp.Tasks, 0)

		if cUrl != baseUrl {
			ts = append(ts, chromedp.Navigate(baseUrl))
		}

		js := "var xhr = new XMLHttpRequest();"
		js = js + fmt.Sprintf("xhr.open('%s', '%s', true);", method, urlStr)

		if method.HasBody() {
			js = js + fmt.Sprintf("xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded; charset=utf-8');")
		}

		//设置header
		for k, v := range headers {
			js = js + fmt.Sprintf("xhr.setRequestHeader('%s', '%s');", k, v)
		}

		if method.HasBody() {
			js = js + fmt.Sprintf("xhr.send('%s');", params)
		} else {
			js = js + fmt.Sprintf("xhr.send();")
		}
		ts = append(ts,
			chromedp.Evaluate(js, nil),
			tag.WaitRequestTaskFinish(&requestId),
		)
		err = tag.RunMain(ts...)
		if err != nil {
			return "", err
		}
	}

	rt := tag.GetRequestTask(requestId)
	if rt == nil {
		return "", errors.New("请求任务为空")
	}

	if !rt.Success() {
		return "", errors.New(rt.ErrorText)
	}
	return rt.ResponseBody, nil
}
