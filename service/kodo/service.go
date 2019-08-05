package kodo

import (
	"fmt"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/client"
	"github.com/qiniu/go-sdk/qiniu/corehandlers"
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/http"
	"github.com/qiniu/go-sdk/qiniu/request"
)

// Kodo 是kodo所有接口，服务的统一入口
type Kodo struct {
	*client.BaseClient
}

const (
	ServiceName = "KODO" // Name of service.)
)

// New 创建一个Kodo指针，是所有Kodo服务的对外统一入口
func New(p client.ConfigProvider, cfgs ...*qiniu.Config) *Kodo {
	c := p.ClientConfig(cfgs...)
	return newClient(*c.Config, c.Handlers)
}

func newClient(cfg qiniu.Config, handlers request.Handlers) *Kodo {
	svc := &Kodo{
		BaseClient: client.New(
			cfg,
			handlers,
		),
	}

	// Handlers
	svc.Handlers.Build.PushBackNamed(corehandlers.BodyHandler)
	svc.Handlers.Unmarshal.PushBackNamed(corehandlers.UnmarshalHandler)

	return svc
}

func (c *Kodo) newRequest(op qiniu.API, params, data interface{}) *request.Request {
	req := c.NewRequest(op, params, data)

	switch op.GetTokenType() {
	case credentials.TokenQiniu:
		req.Handlers.Sign.PushBackNamed(corehandlers.QiniuTokenRequestHandler)
	case credentials.TokenQBox:
		req.Handlers.Sign.PushBackNamed(corehandlers.QboxTokenRequestHandler)

	}

	return req
}

// Stat发起stat请求， 获取存储在七牛空间的文件信息
func (c *Kodo) Stat(bucket, key string) (*FileInfo, error) {
	req, fileInfo := c.StatRequest(bucket, key)
	return fileInfo, req.Send()
}

// StatRequest 返回request.Request指针， 用于发起stat接口请求
func (c *Kodo) StatRequest(bucket, key string) (req *request.Request, info *FileInfo) {
	op := qiniu.NewAPI("POST", fmt.Sprintf("/stat/%s", qiniu.EncodedEntry(bucket, key)), c.Config.RsHost,
		"http", http.CONTENT_TYPE_JSON, credentials.TokenQBox, "stat", ServiceName)
	info = &FileInfo{}
	req = c.newRequest(op, nil, info)
	return
}
