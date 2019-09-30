// Package qiniu 默认会从qiniu.Config, 环境变量， 配置文件获取配置信息，优先级从高到低
// 配置文件是ini格式的文件， 当前可以配置的section有:
// [credentials], [host], [z0], [z1], [z2], [na0], [as0]
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
// [credentials]
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
)

// RequestRetryer 是request.Retryer别名
// 引入该别名防止qiniu和qiniu/request循环依赖
type RequestRetryer interface{}

// Recorder 是kodo.ProgressRecorder的别名
type Recorder interface{}

// Config 为服务客户端提供配置选项
// 默认所有的服务客户端都使用了defaults.DefaultConfig函数返回的默认配置
type Config struct {
	// 开启详细密钥获取错误的详细错误链
	// 只有当想要看到获取密钥过程中的所有错误链是才开启这个选项
	// 默认该值为`false`
	CredentialsChainVerboseErrors *bool

	// 密钥
	Credentials *credentials.Credentials

	// EnforceShouldRetryCheck 在AfterRetryHandler中用来强制性检测请求是否可以重试， 而不管
	// request.Retryable的字段的值
	EnforceShouldRetryCheck *bool

	// 使用的http.Client发送http请求， 默认为http.DefaultClient
	HTTPClient *http.Client

	// LogLevel 是个整型值，代表日志输出的级别， 默认的日志输出级别为LogOff, 不输出日志
	LogLevel *LogLevelType

	// 日志输出接口， 默认输出到标准输出，即stdout
	Logger Logger

	// 请求出错后最大的重试次数, 如果为nil, 那么根据具体的client来配置
	// 默认使用的BaseClient默认发生请求出错有3次重试
	// 当值为0的时候， 没有重试
	MaxRetries *int

	// Retryer 实现了当HTTP请求遇到可以恢复的请求错误的时候的重试逻辑
	//
	// 当为nil或者设置的对象没有实现request.Retryer接口的时候， 默认使用client.DefaultRetryer
	//
	// 当Retyer和MaxRetries都设置了的时候， 前者优先级更高， 后者被忽略
	//
	// 为了确保Retryer字段实现了request.Retryer接口，可以使用帮助函数request.WithRetryer函数
	// 来这只该字段
	//
	//   cfg := request.WithRetryer(api.NewConfig(), myRetryer)
	//
	Retryer RequestRetryer

	// 禁用请求输入的校验
	DisableParamValidation *bool

	// Host 一般都有默认的配置：
	// RsHost： rs.qiniu.com
	// RsfHost: rsf.qiniu.com
	// ApiHost: api.qiniu.com
	// 全局的Host配置
	RsHost *string

	// 七牛RsfHost
	RsfHost *string

	// 七牛API Host
	APIHost *string

	UcHost *string

	// 存储空间所在的区域的名字
	// 支持的区域名字：
	// [`z0`, `z1`, `z2`, `na0`, `as0`]
	// 分别代表`华东`, `华北`, `华南`, `北美`, `东南亚`
	// 如果Region的值不在上面的列表中，那么SDK会忽略该值
	//
	// 如果后续的接口使用的都是一个区域的存储空间，可以设置该值。
	// 比如要操作或者请求服务的存储空间属于不同的存储空间，可以
	// 在具体的接口输入中设置region值，可以覆盖这个地方的配置
	Region *string

	// UploadConcurrency 分片上传的goroutine最大并发上传数量
	// 如果该字段的值<=0或者为nil, 那么使用默认的DefaultUploadConcurrency
	UploadConcurrency *int

	// 是否禁用断点续传， 如果为nil, 那么默认开启
	// 断点续传只适用于上传文件的情况， 如果上传的数据来源于网络或者其他，那么是不支持断点续传的
	DisableResume *bool

	// 分片上传的进度实现, 每当上传成功一块数据，就会调用该接口的Progress()方法
	// 如果该字段为nil或者不是kodo.ProgressRecorder接口的实现， 那么会使用默认的进度实现
	ProgressRecorder Recorder

	// 禁用分片上传的进度条
	DisableRecorder *bool

	// 分片上传的块数达到StoreNumber就保存上传记录信息到本地文件，以实现断点续传
	// 如果设置的值小于等于0， 该字段将被忽略，将使用默认的DefaultStoreNumber值
	StoreNumber *int
}

// NewConfig 返回一个Config指针， 可以使用builder模式设置配置信息
func NewConfig() *Config {
	return &Config{}
}

// WithCredentials 设置密钥信息
func (c *Config) WithCredentials(creds *credentials.Credentials) *Config {
	c.Credentials = creds
	return c
}

// WithDisableRecorder 设置DisableRecorder
func (c *Config) WithDisableRecorder(disable bool) *Config {
	c.DisableRecorder = &disable
	return c
}

// WithStoreNumber 设置StoreNumber
func (c *Config) WithStoreNumber(n int) *Config {
	c.StoreNumber = &n
	return c
}

// WithUploadConcurrency 设置分片上传的最大并发上传可以开启的goroutine数量
func (c *Config) WithUploadConcurrency(concurrency int) *Config {
	c.UploadConcurrency = &concurrency
	return c
}

// WithDisableResume 开启或者关闭断点续传
func (c *Config) WithDisableResume(resume bool) *Config {
	c.DisableResume = &resume
	return c
}

// WithHTTPClient 设置发送请求的HTTPClient
func (c *Config) WithHTTPClient(client *http.Client) *Config {
	c.HTTPClient = client
	return c
}

// WithMaxRetries 设置请求的最大重试次数
func (c *Config) WithMaxRetries(max int) *Config {
	c.MaxRetries = &max
	return c
}

// WithDisableParamValidation 设置DisableParamValidation字段
func (c *Config) WithDisableParamValidation(disable bool) *Config {
	c.DisableParamValidation = &disable
	return c
}

// WithLogLevel 设置日志输出级别LogLevel
func (c *Config) WithLogLevel(level LogLevelType) *Config {
	c.LogLevel = &level
	return c
}

// WithLogger 设置SDK Logger
func (c *Config) WithLogger(logger Logger) *Config {
	c.Logger = logger
	return c
}

// WithRsHost 设置Config.RsHost字段
func (c *Config) WithRsHost(host string) *Config {
	c.RsHost = &host
	return c
}

// WithRsfHost 设置Config.RsfHost字段
func (c *Config) WithRsfHost(host string) *Config {
	c.RsfHost = &host
	return c
}

// WithAPIHost 设置Config.Api字段
func (c *Config) WithAPIHost(host string) *Config {
	c.APIHost = &host
	return c
}

// WithUCHost 设置Config.UcHost字段
func (c *Config) WithUCHost(host string) *Config {
	c.UcHost = &host
	return c
}

// MergeIn 合并传入的cfs信息到c中
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
	if other.HTTPClient != nil {
		dst.HTTPClient = other.HTTPClient
	}
	if other.DisableRecorder != nil {
		dst.DisableRecorder = other.DisableRecorder
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
	if other.StoreNumber != nil {
		dst.StoreNumber = other.StoreNumber
	}
	if other.Retryer != nil {
		dst.Retryer = other.Retryer
	}
	if other.DisableParamValidation != nil {
		dst.DisableParamValidation = other.DisableParamValidation
	}
	if other.RsHost != nil {
		dst.RsHost = other.RsHost
	}
	if other.RsfHost != nil {
		dst.RsfHost = other.RsfHost
	}
	if other.UcHost != nil {
		dst.UcHost = other.UcHost
	}
	if other.APIHost != nil {
		dst.APIHost = other.APIHost
	}
	if other.UploadConcurrency != nil {
		dst.UploadConcurrency = other.UploadConcurrency
	}
	if other.DisableResume != nil {
		dst.DisableResume = other.DisableResume
	}
}

// Copy 合并多个Config到一个Config
func (c *Config) Copy(cfgs ...*Config) *Config {
	dst := &Config{}
	dst.MergeIn(c)

	for _, cfg := range cfgs {
		dst.MergeIn(cfg)
	}

	return dst
}
