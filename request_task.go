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
)

type RequestTask struct {
	RequestId   string           //请求ID
	DocumentUrl string           //文档地址
	Request     *network.Request //请求

	Type network.ResourceType //类型

	IsFinished   bool              //是否已经完成
	Response     *network.Response //响应
	ResponseBody string            //响应体
	ErrorText    string            //错误原因

	HasRewrite    bool                         //是否修改过
	RewriteParams *fetch.ContinueRequestParams //重写参数
}

func (rt *RequestTask) Success() bool {
	return len(rt.ErrorText) < 1
}
