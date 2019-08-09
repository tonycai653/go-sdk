// Package qiniu 默认会从qiniu.Config, 环境变量， 配置文件获取配置信息，优先级从高到低
// 配置文件是ini格式的文件， 当前可以配置的section有:
// [profile], [host], [z0], [z1], [z2], [na0], [as0]
// [profile]提供了账户的信息，比如密钥信息
// [host] 提供了各个接口全局的HOST配置
// [z0], [z1], [z2], [na0], [as0] 是关于存储各个区域的HOST配置
// 可以配置的信息有UpHosts(可用的上传域名列表), CdnUpHosts（可用的加速上传域名列表),
// 各个区域对应的RsHost, RsfHost, ApiHost, IoHost（存储下载入口)
//
// 如果在配置文件中，同时配置了全局的Host和区域Host, 那么会使用区域的配置
//
//该文件的格式如下:
//
// [profile]
// QINIU_ACCESS_KEY = ""
// QINIU_SECRET_ACESS_KEY_ID = ""
//
// [host]
// QINIU_RS_HOST = ""
// QINIU_RSF_HOST = ""
// QINIU_UC_HOST = ""
// QINIU_API_HOST = ""
//
// [z0]
// UpHosts = "domain1,domain2"
// CdnUpHosts = "domain1,domain2"
// RsHost = "",
// RsfHost = "",
// ApiHost = "",
// IoHost = "",
package qiniu

import (
	"net/http"

	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/definitions"
)

// UseServiceDefaultRetries instructs the config to use the service's own
// default number of retries. This will be the default action if
// Config.MaxRetries is nil also.
const UseServiceDefaultRetries = -1

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
	PB = 1024 * GB
)

const (
	// 默认的RsHost
	DefaultRsHost = "rs.qiniu.com"

	// 默认的RsfHost
	DefaultRsfHost = "rsf.qiniu.com"

	// 默认的APIHost
	DefaultAPIHost = "api.qiniu.com"

	// 查询存储空间相关域名
	DefaultUcHost = "uc.qbox.me"

	// 默认的最大的可以使用表单方式上传的文件大小
	DefaultFormSize = 1 * MB
)

// RequestRetryer is an alias for a type that implements the request.Retryer
// interface.
type RequestRetryer interface{}

// Config 为服务客户端提供配置选项
// 默认所有的服务客户端都使用了defaults.DefaultConfig函数返回的默认配置
type Config struct {
	// 开启详细密钥获取错误的详细错误链
	// 只有当想要看到获取密钥过程中的所有错误链是才开启这个选项
	// 默认该值为`false`
	CredentialsChainVerboseErrors *bool

	// 密钥
	Credentials *credentials.Credentials

	// EnforceShouldRetryCheck is used in the AfterRetryHandler to always call
	// ShouldRetry regardless of whether or not if request.Retryable is set.
	// This will utilize ShouldRetry method of custom retryers. If EnforceShouldRetryCheck
	// is not set, then ShouldRetry will only be called if request.Retryable is nil.
	// Proper handling of the request.Retryable field is important when setting this field.
	EnforceShouldRetryCheck *bool

	// Set this to `true` to disable SSL when sending requests. Defaults
	// to `false`.
	DisableSSL *bool

	// The HTTP client to use when sending requests. Defaults to
	// `http.DefaultClient`.
	HTTPClient *http.Client

	// An integer value representing the logging level. The default log level
	// is zero (LogOff), which represents no logging. To enable logging set
	// to a LogLevel Value.
	LogLevel *LogLevelType

	// The logger writer interface to write logging messages to. Defaults to
	// standard out.
	Logger Logger

	// The maximum number of times that a request will be retried for failures.
	// Defaults to -1, which defers the max retry setting to the service
	// specific configuration.
	MaxRetries *int

	// Retryer guides how HTTP requests should be retried in case of
	// recoverable failures.
	//
	// When nil or the value does not implement the request.Retryer interface,
	// the client.DefaultRetryer will be used.
	//
	// When both Retryer and MaxRetries are non-nil, the former is used and
	// the latter ignored.
	//
	// To set the Retryer field in a type-safe manner and with chaining, use
	// the request.WithRetryer helper function:
	//
	//   cfg := request.WithRetryer(api.NewConfig(), myRetryer)
	//
	Retryer RequestRetryer

	// Disables semantic parameter validation, which validates input for
	// missing required fields and/or other semantic request input errors.
	DisableParamValidation *bool

	// Instructs the endpoint to be generated for a service client to
	// be the dual stack endpoint. The dual stack endpoint will support
	// both IPv4 and IPv6 addressing.
	//
	// Setting this for a service which does not support dual stack will fail
	// to make requets. It is not recommended to set this value on the session
	// as it will apply to all service clients created with the session. Even
	// services which don't support dual stack endpoints.
	UseDualStack *bool

	// 指示当dump http response的时候是否输出body
	// LogDebugBody = true 的时候输出body
	// 否则不输出body
	LogDebugHTTPRequestBody bool

	LogDebugHTTPResponseBody bool

	// Host 一般都有默认的配置：
	// RsHost： rs.qiniu.com
	// RsfHost: rsf.qiniu.com
	// ApiHost: api.qiniu.com
	// 全局的Host配置
	RsHost string

	// 七牛RsfHost
	RsfHost string

	// 七牛API Host
	ApiHost string

	UcHost string

	// 存储空间所在的区域的名字
	// 支持的区域名字：
	// [`z0`, `z1`, `z2`, `na0`, `as0`]
	// 分别代表`华东`, `华北`, `华南`, `北美`, `东南亚`
	// 如果Region的值不在上面的列表中，那么SDK会忽略该值
	//
	// 如果后续的接口使用的都是一个区域的存储空间，可以设置该值。
	// 比如要操作或者请求服务的存储空间属于不同的存储空间，可以
	// 在具体的接口输入中设置region值，可以覆盖这个地方的配置
	Region string

	// 各个区域的HOSTS， 如果全局配置和各个区域的HOST配置都存在
	// 那么会使用各个存储区域的HOST配置

	// 华东
	RegionHD definitions.Host

	// 华南
	RegionHN definitions.Host

	// 华北
	RegionHB definitions.Host

	// 东南亚
	RegionAsia definitions.Host

	// 北美
	RegionNA definitions.Host
}

// NewConfig returns a new Config pointer that can be chained with builder
// methods to set multiple configuration values inline without using pointers.
func NewConfig() *Config {
	return &Config{}
}

// WithCredentials sets a config Credentials value returning a Config pointer
// for chaining.
func (c *Config) WithCredentials(creds *credentials.Credentials) *Config {
	c.Credentials = creds
	return c
}

// WithDisableSSL sets a config DisableSSL value returning a Config pointer
// for chaining.
func (c *Config) WithDisableSSL(disable bool) *Config {
	c.DisableSSL = &disable
	return c
}

// WithHTTPClient sets a config HTTPClient value returning a Config pointer
// for chaining.
func (c *Config) WithHTTPClient(client *http.Client) *Config {
	c.HTTPClient = client
	return c
}

// WithMaxRetries sets a config MaxRetries value returning a Config pointer
// for chaining.
func (c *Config) WithMaxRetries(max int) *Config {
	c.MaxRetries = &max
	return c
}

// WithDisableParamValidation sets a config DisableParamValidation value
// returning a Config pointer for chaining.
func (c *Config) WithDisableParamValidation(disable bool) *Config {
	c.DisableParamValidation = &disable
	return c
}

// WithLogLevel sets a config LogLevel value returning a Config pointer for
// chaining.
func (c *Config) WithLogLevel(level LogLevelType) *Config {
	c.LogLevel = &level
	return c
}

// WithLogger sets a config Logger value returning a Config pointer for
// chaining.
func (c *Config) WithLogger(logger Logger) *Config {
	c.Logger = logger
	return c
}

// WithUseDualStack sets a config UseDualStack value returning a Config
// pointer for chaining.
func (c *Config) WithUseDualStack(enable bool) *Config {
	c.UseDualStack = &enable
	return c
}

// WithLogDebugHttpRequestBody 开启输出http 请求body选项
func (c *Config) WithLogDebugHttpRequestBody(enable bool) *Config {
	c.LogDebugHTTPRequestBody = enable
	return c
}

// WithLogDebugHttpResponseBody开启输出http响应body选项
func (c *Config) WithLogDebugHttpResponseBody(enable bool) *Config {
	c.LogDebugHTTPResponseBody = enable
	return c
}

// WithRsHost 设置Config.RsHost字段
func (c *Config) WithRsHost(host string) *Config {
	c.RsHost = host
	return c
}

// WithRsfHost 设置Config.RsfHost字段
func (c *Config) WithRsfHost(host string) *Config {
	c.RsfHost = host
	return c
}

// WithAPIHost 设置Config.Api字段
func (c *Config) WithAPIHost(host string) *Config {
	c.ApiHost = host
	return c
}

// WithUCHost 设置Config.UcHost字段
func (c *Config) WithUCHost(host string) *Config {
	c.UcHost = host
	return c
}

// MergeIn merges the passed in configs into the existing config object.
func (c *Config) MergeIn(cfgs ...*Config) {
	for _, other := range cfgs {
		mergeInConfig(c, other)
	}
}

func mergeInConfig(dst *Config, other *Config) {
	if other == nil {
		return
	}

	if other.Credentials != nil {
		dst.Credentials = other.Credentials
	}

	if other.DisableSSL != nil {
		dst.DisableSSL = other.DisableSSL
	}

	if other.HTTPClient != nil {
		dst.HTTPClient = other.HTTPClient
	}

	if other.LogLevel != nil {
		dst.LogLevel = other.LogLevel
	}

	if other.Logger != nil {
		dst.Logger = other.Logger
	}

	if other.MaxRetries != nil {
		dst.MaxRetries = other.MaxRetries
	}

	if other.Retryer != nil {
		dst.Retryer = other.Retryer
	}

	if other.DisableParamValidation != nil {
		dst.DisableParamValidation = other.DisableParamValidation
	}

	if other.UseDualStack != nil {
		dst.UseDualStack = other.UseDualStack
	}
	if other.RsHost != "" {
		dst.RsHost = other.RsHost
	}
	if other.RsfHost != "" {
		dst.RsfHost = other.RsfHost
	}
	if other.UcHost != "" {
		dst.UcHost = other.UcHost
	}
	if other.ApiHost != "" {
		dst.ApiHost = other.ApiHost
	}
	dst.LogDebugHTTPRequestBody = other.LogDebugHTTPRequestBody
	dst.LogDebugHTTPResponseBody = other.LogDebugHTTPResponseBody
}

// Copy will return a shallow copy of the Config object. If any additional
// configurations are provided they will be merged into the new config returned.
func (c *Config) Copy(cfgs ...*Config) *Config {
	dst := &Config{}
	dst.MergeIn(c)

	for _, cfg := range cfgs {
		dst.MergeIn(cfg)
	}

	return dst
}
