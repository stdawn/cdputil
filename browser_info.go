/**
 * @Time: 2023/11/30 10:22
 * @Author: LiuKun
 * @File: browser_info.go
 * @Description:
 */

package cdputil

import "sync"

var (
	rtkOnce          sync.Once
	shareBrowserInfo *BrowserInfo
)

// GetBrowserInfo 线程安全的获取浏览器程序信息单例
func GetBrowserInfo() *BrowserInfo {
	rtkOnce.Do(func() {
		shareBrowserInfo = new(BrowserInfo)
		shareBrowserInfo.name = "chrome.exe"
		shareBrowserInfo.url = "$ProgramFiles (x86)\\Google\\Chrome\\Application\\chrome.exe"
		shareBrowserInfo.port = 9222
		shareBrowserInfo.windowWidth = 0
		shareBrowserInfo.windowHeight = 0
		shareBrowserInfo.headless = false
	})
	return shareBrowserInfo
}

// BrowserInfo 浏览器程序信息
type BrowserInfo struct {
	name         string // 浏览器进程名称
	url          string // 浏览器程序启动地址
	port         uint   // 浏览器程序CDP端口
	windowWidth  uint   // 浏览窗口默认宽度
	windowHeight uint   // 浏览窗口默认高度
	headless     bool   // 是否隐藏浏览器窗口
}

// WithName 设置浏览器名称， 默认为chrome.exe
func (b *BrowserInfo) WithName(name string) *BrowserInfo {
	b.name = name
	return b
}

// WithUrl 设置浏览器启动地址， 默认为$ProgramFiles (x86)\\Google\\Chrome\\Application\\chrome.exe
func (b *BrowserInfo) WithUrl(url string) *BrowserInfo {
	b.url = url
	return b
}

// WithPort 设置浏览器CDP端口， 默认为9222
func (b *BrowserInfo) WithPort(port uint) *BrowserInfo {
	b.port = port
	return b
}

// WithWindowWidth 设置浏览器窗口默认宽度， 默认为0,
func (b *BrowserInfo) WithWindowWidth(width uint) *BrowserInfo {
	b.windowWidth = width
	return b
}

// WithWindowHeight 设置浏览器窗口默认高度， 默认为0
func (b *BrowserInfo) WithWindowHeight(height uint) *BrowserInfo {
	b.windowHeight = height
	return b
}

// WithHeadless 设置是否隐藏浏览器窗口， 默认为false
func (b *BrowserInfo) WithHeadless(headless bool) *BrowserInfo {
	b.headless = headless
	return b
}
