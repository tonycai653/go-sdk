package client

import (
	"fmt"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/request"
)

// Config 为服务客户端提供配置信息
type Config struct {
	Config   *qiniu.Config
	Handlers request.Handlers
}

// ConfigProvider provides a generic way for a service client to receive
// the ClientConfig without circular dependencies.
type ConfigProvider interface {
	ClientConfig(cfgs ...*qiniu.Config) Config
}

// BaseClient 实现了请求和影响处理的基础逻辑
type BaseClient struct {
	request.Retryer

	Config   qiniu.Config
	Handlers request.Handlers
}

// New 返回一个BaseClient指针
func New(cfg qiniu.Config, handlers request.Handlers, options ...func(*BaseClient)) *BaseClient {
	svc := &BaseClient{
		Config:   cfg,
		Handlers: handlers.Copy(),
	}

	switch retryer, ok := cfg.Retryer.(request.Retryer); {
	case ok:
		svc.Retryer = retryer
	case cfg.Retryer != nil && cfg.Logger != nil:
		s := fmt.Sprintf("WARNING: %T does not implement request.Retryer; using DefaultRetryer instead", cfg.Retryer)
		cfg.Logger.Log(s)
		fallthrough
	default:
		if cfg.MaxRetries == nil {
			cfg.MaxRetries = qiniu.Int(3)
		}
		svc.Retryer = DefaultRetryer{NumMaxRetries: qiniu.IntValue(cfg.MaxRetries)}
	}

	svc.AddDebugHandlers()

	for _, option := range options {
		option(svc)
	}

	return svc
}

// NewRequest 返回一个request.Request指针
func (c *BaseClient) NewRequest(operation *request.API, params interface{}, data interface{}) *request.Request {
	return request.New(c.Config, c.Handlers, c.Retryer, operation, params, data)
}

// AddDebugHandlers 注册打印请求和响应的处理函数
func (c *BaseClient) AddDebugHandlers() {
	if !c.Config.LogLevel.AtLeast(qiniu.LogDebug) {
		return
	}

	c.Handlers.Send.PushFrontNamed(LogHTTPRequestHandler)
	c.Handlers.Send.PushBackNamed(LogHTTPResponseHandler)
}
