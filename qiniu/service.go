package qiniu

import (
	"net/http"
	"strings"

	"github.com/qiniu/go-sdk/qiniu/credentials"
)

// API 是一个接口
// 所有向七牛后段服务提交的请求都要实现该接口
type API interface {
	// 返回请求接口的Header
	GetHeaders() http.Header

	// 接口请求URL
	GetURL() string

	// 请求接口的方法, POST, PUT, GET etc
	GetMethod() string

	GetTokenType() credentials.TokenType

	// 接口名称
	Name() string

	// 服务名称KODO, CDN, DORA, etc
	ServiceName() string
}

// BaseAPI 实现API接口, 实现基本常用的接口请求行为
type BaseAPI struct {
	Path        string
	Method      string
	Host        string
	ContentType string
	Scheme      string
	TokenType   credentials.TokenType

	// 服务名字
	SName string

	// 接口名字
	APIName string
}

// NewAPI 返回一个BaseAPI指针
func NewAPI(method, path, host, scheme, contentType string, tokenType credentials.TokenType, apiName, service string) *BaseAPI {
	if method == "" {
		method = "GET"
	}
	if scheme == "" {
		scheme = "http"
	}
	path = checkFields(path, "path")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	host = checkFields(host, "host")
	contentType = checkFields(contentType, "content-type")

	return &BaseAPI{
		Path:        path,
		Method:      method,
		Host:        host,
		ContentType: contentType,
		Scheme:      scheme,
		TokenType:   tokenType,
		SName:       service,
		APIName:     apiName,
	}
}

func checkFields(field string, fieldName string) string {
	s := strings.TrimSpace(field)
	if s == "" {
		panic("API " + fieldName + " cannot be empty")
	}
	return s
}

// GetHeaders 返回请求接口的http Header
func (b *BaseAPI) GetHeaders() http.Header {
	header := make(http.Header)
	header.Set("Content-Type", b.ContentType)
	header.Set("Host", b.Host)
	return header
}

// GetURL 返回请求接口的URL
func (b *BaseAPI) GetURL() string {
	return strings.Join([]string{b.Scheme, "://", b.Host, b.Path}, "")
}

// GetMethod 返回请求的方法
func (b *BaseAPI) GetMethod() string {
	return b.Method
}

// GetTokenType 返回签名的类型
func (b *BaseAPI) GetTokenType() credentials.TokenType {
	return b.TokenType
}

// Name 返回接口名称
func (b *BaseAPI) Name() string {
	return b.APIName
}

// ServiceName 返回服务的名称， KODO， CDN, DORA, etc
func (b *BaseAPI) ServiceName() string {
	return b.SName
}
