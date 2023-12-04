/**
 * @Time: 2023/11/22 14:55
 * @Author: LiuKun
 * @File: cdp_util.go
 * @Description:
 */

package cdputil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
	nt "github.com/stdawn/network"
	"github.com/stdawn/util"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	remoteContext   context.Context
	remoteCancel    context.CancelFunc
	remoteContextMu sync.Mutex
)

// HeaderConverter network.Headers转为[]*fetch.HeaderEntry
func HeaderConverter(header network.Headers) []*fetch.HeaderEntry {
	hs := make([]*fetch.HeaderEntry, 0)
	for k, v := range header {
		hs = append(hs, &fetch.HeaderEntry{Name: k, Value: v.(string)})
	}
	return hs
}

// CookieConverter []*network.Cookie转为[]*http.Cookie
func CookieConverter(cookies []*network.Cookie) []*http.Cookie {
	cs := make([]*http.Cookie, 0)
	for _, cookie := range cookies {

		c := &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			HttpOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
			Expires:  time.Unix(int64(cookie.Expires), 0),
		}

		lowerVal := strings.ToLower(cookie.SameSite.String())
		switch lowerVal {
		case "lax":
			c.SameSite = http.SameSiteLaxMode
		case "strict":
			c.SameSite = http.SameSiteStrictMode
		case "none":
			c.SameSite = http.SameSiteNoneMode
		default:
			c.SameSite = http.SameSiteDefaultMode
		}
		cs = append(cs, c)
	}
	return cs
}

// GetCookies 获取特定域名的Cookies
func GetCookies(ctx context.Context, domain string) ([]*http.Cookie, error) {
	cookies, err := storage.GetCookies().Do(ctx)
	if err != nil {
		return nil, err
	}

	newCookies := make([]*network.Cookie, 0)
	for _, cookie := range cookies {
		if cookie.Domain == domain {
			newCookies = append(newCookies, cookie)
		}
	}

	cs := CookieConverter(newCookies)
	return cs, nil
}

// 清除远程上下文
func clearRemoteContext(c context.Context) {
	remoteContextMu.Lock()
	defer remoteContextMu.Unlock()
	if c != remoteContext {
		return
	}
	if remoteCancel != nil {
		remoteCancel()
	}
	remoteContext = nil
	remoteCancel = nil
}

// 获取远程上下文
func getRemoteContext() (context.Context, context.CancelFunc, error) {
	remoteContextMu.Lock()
	defer remoteContextMu.Unlock()

	devtoolsWsUrl := getDevtoolsWsUrl()

	if len(devtoolsWsUrl) > 0 {
		if remoteContext != nil {
			return remoteContext, remoteCancel, nil
		}
		remoteContext, remoteCancel = chromedp.NewRemoteAllocator(context.Background(), devtoolsWsUrl)
		return remoteContext, remoteCancel, nil
	}

	if util.IsExeRunning(GetBrowserInfo().name) {
		//关闭浏览器
		err0 := util.CloseProgram(GetBrowserInfo().name)
		if err0 != nil {
			return nil, nil, errors.New(fmt.Sprintf("close original browser (%s) error:%s", GetBrowserInfo().name, err0.Error()))
		}
		time.Sleep(time.Second)
	}

	//指定cdp的ws连接端口
	agr := []string{
		fmt.Sprintf("--remote-debugging-port=%d", GetBrowserInfo().port),
	}

	if GetBrowserInfo().headless {
		//指定浏览器无头模式
		agr = append(agr, "--headless")
	} else {
		//指定浏览器窗口大小
		if GetBrowserInfo().windowHeight > 0 && GetBrowserInfo().windowWidth > 0 {
			agr = append(agr, fmt.Sprintf("--window-size=%d,%d", GetBrowserInfo().windowWidth, GetBrowserInfo().windowHeight))
		}
	}

	//启动浏览器
	cmd := exec.Command(os.ExpandEnv(GetBrowserInfo().url), agr...)
	err1 := cmd.Start()
	if err1 != nil {
		return nil, nil, errors.New(fmt.Sprintf("start browser (%s) error : %s", GetBrowserInfo().url, err1.Error()))
	}

	//等待浏览器启动完成
	for {
		time.Sleep(100 * time.Millisecond)
		devtoolsWsUrl = getDevtoolsWsUrl()
		if len(devtoolsWsUrl) > 0 {
			break
		}
	}

	remoteContext, remoteCancel = chromedp.NewRemoteAllocator(context.Background(), devtoolsWsUrl)
	return remoteContext, remoteCancel, nil

}

// 获取devtools的ws连接地址
func getDevtoolsWsUrl() string {
	res, err := nt.Request(nt.RequestMethodGet, fmt.Sprintf("http://localhost:%d/json/version", GetBrowserInfo().port), "", nil)
	if err == nil {
		resMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(res), &resMap)
		if err == nil {
			return util.GetStringFromMap(resMap, "webSocketDebuggerUrl")
		}
	}
	return ""
}
