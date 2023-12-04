/**
 * @Time: 2023/11/27 14:37
 * @Author: LiuKun
 * @File: request_task.go
 * @Description:
 */

package cdputil

import (
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"time"
)

type RequestTask struct {
	RequestId        string           //请求ID
	DocumentUrl      string           //文档地址
	Request          *network.Request //请求
	RequestStartTime time.Time        //请求时间

	Type network.ResourceType //类型

	IsFinished   bool              //是否已经完成
	Response     *network.Response //响应
	ResponseBody string            //响应体
	ErrorText    string            //错误原因

	HasRewrite    bool                         //是否修改过
	RewriteParams *fetch.ContinueRequestParams //重写参数
}

// Success 是否成功
func (rt *RequestTask) Success() bool {
	return len(rt.ErrorText) < 1
}

// IsTimeout 是否超时
func (rt *RequestTask) IsTimeout(timeout time.Duration) bool {
	return time.Now().Sub(rt.RequestStartTime) >= timeout
}
