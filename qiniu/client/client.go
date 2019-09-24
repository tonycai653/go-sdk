package client

import (
	"fmt"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/request"
)

// A Config provides configuration to a service client instance.
type Config struct {
	Config   *qiniu.Config
	Handlers request.Handlers
}

// ConfigProvider provides a generic way for a service client to receive
// the ClientConfig without circular dependencies.
type ConfigProvider interface {
	ClientConfig(cfgs ...*qiniu.Config) Config
}

// A Client implements the base client request and response handling
// used by all service clients.
type BaseClient struct {
	request.Retryer

	Config   qiniu.Config
	Handlers request.Handlers
}

// New will return a pointer to a new initialized service client.
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
		maxRetries := qiniu.IntValue(cfg.MaxRetries)
		if cfg.MaxRetries == nil || maxRetries == qiniu.UseServiceDefaultRetries {
			maxRetries = 3
		}
		svc.Retryer = DefaultRetryer{NumMaxRetries: maxRetries}
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
