package corehandlers

import (
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/qerr"
	"github.com/qiniu/go-sdk/qiniu/request"
)

// QboxTokenRequestHandler 给请求加上Authorization 请求头, 使用Qbox签方式.
// 不同的接口要求的签名方式不一样， 有个需要Qbox token, 有的需要Qiniu Token
var QboxTokenRequestHandler = request.NamedHandler{
	Name: "qiniusdk.auth.QboxTokenRequestHandler",
	Fn: func(r *request.Request) {
		v, err := r.Config.Credentials.Get()
		if err != nil {
			r.Error = qerr.New(credentials.ErrCredsRetrieve, "failed to retrieve credential value", err)
			return
		}
		token, err := v.SignRequest(r.HTTPRequest)
		if err != nil {
			r.Error = qerr.New(credentials.ErrSignRequest, "sign request error", err)
			return
		}
		r.HTTPRequest.Header.Add("Authorization", "QBox "+token)
	},
}

// QboxTokenRequestHandler 给请求加上Authorization 请求头, 使用Qiniu签名方式
// 不同的接口要求的签名方式不一样， 有个需要Qbox token, 有的需要Qiniu Token
var QiniuTokenRequestHandler = request.NamedHandler{
	Name: "qiniusdk.auth.QiniuTokenRequestHandler",
	Fn: func(r *request.Request) {
		v, err := r.Config.Credentials.Get()
		if err != nil {
			r.Error = qerr.New(credentials.ErrCredsRetrieve, "failed to retrieve credential value", err)
			return
		}
		token, err := v.SignRequestV2(r.HTTPRequest)
		if err != nil {
			r.Error = qerr.New(credentials.ErrSignRequest, "sign request error", err)
			return
		}
		r.HTTPRequest.Header.Add("Authorization", "Qiniu "+token)
	},
}
