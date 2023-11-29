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
	"github.com/chromedp/chromedp"
	nt "github.com/stdawn/network"
	"github.com/stdawn/util"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	// 浏览器进程名称
	browserProgramName = "chrome.exe"
	// 浏览器程序启动地址
	browserProgramUrl = "$ProgramFiles (x86)\\Google\\Chrome\\Application\\chrome.exe"
	// 浏览器程序CDP端口
	browserProgramPort = 9222
)

var (
	remoteContext   context.Context
	remoteCancel    context.CancelFunc
	remoteContextMu sync.Mutex
)

// SetBrowserProgramInfo 设置浏览器程序信息
// name: 浏览器进程名称， 默认为chrome.exe
// url: 浏览器程序启动地址， 默认为%ProgramFiles(x86)%\Google\Chrome\Application\chrome.exe
// port: 浏览器程序CDP端口，默认为9222
func SetBrowserProgramInfo(name string, url string, port int) {
	browserProgramName = name
	browserProgramUrl = url
	browserProgramPort = port
}

// HeaderConverter network.Headers转为[]*fetch.HeaderEntry
func HeaderConverter(header network.Headers) []*fetch.HeaderEntry {
	hs := make([]*fetch.HeaderEntry, 0)
	for k, v := range header {
		hs = append(hs, &fetch.HeaderEntry{Name: k, Value: v.(string)})
	}
	return hs
}

// 获取远程上下文
func getRemoteContext() (context.Context, context.CancelFunc, error) {
	remoteContextMu.Lock()
	defer remoteContextMu.Unlock()
	
	devtoolsWsUrl := ""

	res, err := nt.Request(nt.RequestMethodGet, fmt.Sprintf("http://localhost:%d/json/version", browserProgramPort), "", nil)
	if err == nil {
		resMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(res), &resMap)
		if err == nil {
			wsUrl := util.GetStringFromMap(resMap, "webSocketDebuggerUrl")
			if len(wsUrl) < 1 {
				err = errors.New("get webSocketDebuggerUrl error")
			} else {
				devtoolsWsUrl = wsUrl
			}
		}
	}

	if len(devtoolsWsUrl) < 1 {
		//打开浏览器
		err1 := util.CloseProgram(browserProgramName)
		if err1 != nil {
			return nil, nil, errors.New(fmt.Sprintf("close original browser (%s) error:%s", browserProgramName, err1.Error()))
		}

		time.Sleep(time.Second)

		cmd := exec.Command(os.ExpandEnv(browserProgramUrl), fmt.Sprintf("--remote-debugging-port=%d", browserProgramPort))
		err1 = cmd.Start()
		if err1 != nil {
			return nil, nil, errors.New(fmt.Sprintf("start browser (%s) error : %s", browserProgramUrl, err1.Error()))
		}

		devtoolsWsUrl = fmt.Sprintf("ws://localhost:%d", browserProgramPort)
	}

	if remoteContext != nil {
		return remoteContext, remoteCancel, nil
	}

	remoteContext, remoteCancel = chromedp.NewRemoteAllocator(context.Background(), devtoolsWsUrl)
	return remoteContext, remoteCancel, nil

}
