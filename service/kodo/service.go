package kodo

import (
	"fmt"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/client"
	"github.com/qiniu/go-sdk/qiniu/corehandlers"
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/defs"
	"github.com/qiniu/go-sdk/qiniu/qerr"
	"github.com/qiniu/go-sdk/qiniu/request"
)

// Kodo 是kodo所有接口，服务的统一入口
type Kodo struct {
	*client.BaseClient
}

const (
	// ServiceName 是存储服务的名字
	ServiceName = "KODO"
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

func (c *Kodo) newRequest(op *request.API, params, data interface{}) *request.Request {
	req := c.NewRequest(op, params, data)

	switch op.TokenType {
	case credentials.TokenQiniu:
		req.Handlers.Sign.PushBackNamed(corehandlers.QiniuTokenRequestHandler)
	case credentials.TokenQBox:
		req.Handlers.Sign.PushBackNamed(corehandlers.QboxTokenRequestHandler)
	}

	return req
}

// QueryRegionDomains 通过请求API获取bucket所在存储区域的下载和上传入口域名组信息
// 返回一个BucketIoUpDomains指针
func (c *Kodo) QueryRegionDomains(bucket string) (*RegionDomains, error) {
	req, domains, err := c.QueryRegionDomainsRequest(bucket)
	if err != nil {
		return nil, err
	}
	return domains, req.Send()
}

// QueryRegionDomainsRequest 返回一个request.Request指针， 用于向v3/query接口请求存储空间所在区域的下载和上传域名组信息
func (c *Kodo) QueryRegionDomainsRequest(bucket string) (req *request.Request, domains *RegionDomains, err error) {
	v, gerr := c.Config.Credentials.Get()
	if gerr != nil {
		err = qerr.New(credentials.ErrCredsRetrieve, "failed to get credentials value", gerr)
		return
	}
	op := &request.API{
		Scheme:      "http",
		Path:        fmt.Sprintf("/v3/query?ak=%s&bucket=%s", v.AccessKey, bucket),
		Method:      "GET",
		Host:        *c.Config.UcHost,
		ContentType: defs.CONTENT_TYPE_FORM,
		APIName:     "v3query",
		ServiceName: ServiceName,
	}
	domains = &RegionDomains{}
	req = c.newRequest(op, nil, domains)
	return
}
