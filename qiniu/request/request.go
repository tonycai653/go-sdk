package request

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/qerr"
)

const (
	// ErrCodeSerialization 序列化过程中发生错误
	ErrCodeSerialization = "SerializationError"

	// ErrCodeRead 读取http数据的时候发生错误
	ErrCodeRead = "ReadError"

	// ErrCodeResponseTimeout 读取网络数据响应超时
	ErrCodeResponseTimeout = "ResponseTimeout"

	// ErrCodeCanceled http请求被取消
	ErrCodeCanceled = "RequestCanceled"
)

// Request 是发送到服务端的请求
type Request struct {
	Config   qiniu.Config
	Handlers Handlers

	ServiceName string

	Retryer
	AttemptTime            time.Time
	Time                   time.Time
	Api                    *API
	HTTPRequest            *http.Request
	HTTPResponse           *http.Response
	Body                   io.ReadSeeker
	BodyStart              int64 // 读取数据的开始位置
	Params                 interface{}
	Error                  error
	Data                   interface{}
	RequestID              string
	RetryCount             int
	Retryable              *bool
	RetryDelay             time.Duration
	SignedHeaderVals       http.Header
	DisableFollowRedirects bool

	context context.Context

	built bool

	safeBody *offsetReader
}

// API 封装了向七牛API发起请求要用到的信息
type API struct {
	Path        string
	Method      string
	Host        string
	ContentType string
	Scheme      string
	TokenType   credentials.TokenType

	// 服务名字
	ServiceName string

	// 接口名字
	APIName string
}

// Name 返回服务端API的名字
func (a *API) Name() string {
	return a.APIName
}

// URL 返回请求的URL地址
//
// 如果API的Host带了请求的scheme, 那么使用该sheme
// 如果HOST没有带scheme, 并且Scheme字段不为空， 那么使用Scheme字段作为scheme
// 上述两个条件都不满足，则默认使用http作为scheme
func (a *API) URL() string {
	var (
		scheme string
		host   string
	)

	if strings.Contains(a.Host, "http://") || strings.Contains(a.Host, "https://") {
		splits := strings.SplitN(a.Host, "://", 2)
		scheme = splits[0]
		host = splits[1]
	} else {
		scheme = a.Scheme
		host = a.Host
	}
	if scheme == "" {
		scheme = "http"
	}
	host = strings.TrimRight(host, "/")
	path := strings.TrimLeft(a.Path, "/")

	return strings.Join([]string{scheme, "://", host, "/", path}, "")
}

// New 返回Request指针， 用于发起API请求
// Params 是Request的Body部分， 发出的http请求的body部分
// Data 是http响应的响应体反序列化到的结构体
func New(cfg qiniu.Config, handlers Handlers, retryer Retryer,
	operation *API, params interface{}, data interface{}) *Request {

	method := operation.Method
	if method == "" {
		method = "POST"
	}
	contentType := operation.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	httpReq, _ := http.NewRequest(method, "", nil)
	httpReq.Header.Set("Content-Type", contentType)

	var err error
	httpReq.URL, err = url.Parse(operation.URL())
	if err != nil {
		httpReq.URL = &url.URL{}
		err = qerr.New("InvalidEndpointURL", "invalid endpoint uri", err)
	}

	SanitizeHostForHeader(httpReq)

	r := &Request{
		Config:   cfg,
		Handlers: handlers.Copy(),

		Retryer:     retryer,
		Time:        time.Now(),
		Api:         operation,
		HTTPRequest: httpReq,
		Body:        nil,
		Params:      params,
		Error:       err,
		Data:        data,
		ServiceName: operation.ServiceName,
	}
	r.SetBufferBody([]byte{})

	return r
}

// Option 封装了一个函数，用来修改或者设置请求的字段
type Option func(*Request)

// WithGetResponseHeader 构建一个Option, 用来从Response中获取一个请求头的值
func WithGetResponseHeader(key string, val *string) Option {
	return func(r *Request) {
		r.Handlers.Complete.PushBack(func(req *Request) {
			*val = req.HTTPResponse.Header.Get(key)
		})
	}
}

// WithGetResponseHeaders 构建一个请求Option，用来获取所有的请求头
func WithGetResponseHeaders(headers *http.Header) Option {
	return func(r *Request) {
		r.Handlers.Complete.PushBack(func(req *Request) {
			*headers = req.HTTPResponse.Header
		})
	}
}

// WithGetResponseStatusCode 构建一个请求Option, 获取响应状态码
func WithGetResponseStatusCode(statusCode *int) Option {
	return func(r *Request) {
		r.Handlers.Complete.PushBack(func(req *Request) {
			*statusCode = req.HTTPResponse.StatusCode
		})
	}
}

// WithLogLevel 构建一个请求Option, 用来设置请求的日志级别
func WithLogLevel(l qiniu.LogLevelType) Option {
	return func(r *Request) {
		r.Config.LogLevel = &l
	}
}

// ApplyOptions 会依次应用选项到请求上
func (r *Request) ApplyOptions(opts ...Option) {
	for _, opt := range opts {
		opt(r)
	}
}

// Context 总是返回非nil的Context, 如果请求没有设置context, 那么返回context.Background context
func (r *Request) Context() context.Context {
	if r.context != nil {
		return r.context
	}
	return context.Background()
}

// SetContext 设置请求的Context, 用来取消发送中的请求。Context不能为nil, 否则该方法会直接panic
func (r *Request) SetContext(ctx context.Context) {
	if ctx == nil {
		panic("context cannot be nil")
	}
	r.context = ctx
	r.HTTPRequest = r.HTTPRequest.WithContext(ctx)
}

// WillRetry 返回请求是否可以重试
func (r *Request) WillRetry() bool {
	if !qiniu.IsReaderSeekable(r.Body) && r.HTTPRequest.Body != http.NoBody {
		return false
	}
	return r.Error != nil && qiniu.BoolValue(r.Retryable) && r.RetryCount < r.MaxRetries()
}

func fmtAttemptCount(retryCount, maxRetries int) string {
	return fmt.Sprintf("attempt %v/%v", retryCount, maxRetries)
}

// ParamsFilled 返回true， 如果r.Params请求参数不为nil, 并且是有效的，否则返回false
func (r *Request) ParamsFilled() bool {
	return r.Params != nil && reflect.ValueOf(r.Params).Elem().IsValid()
}

// DataFilled 返回true, 如果r.Data不为nil, 并且是有效的, 否则返回false
func (r *Request) DataFilled() bool {
	return r.Data != nil && reflect.ValueOf(r.Data).Elem().IsValid()
}

// SetBufferBody 使用字节切片设置请求body
func (r *Request) SetBufferBody(buf []byte) {
	r.SetReaderBody(bytes.NewReader(buf))
}

// SetStringBody 使用字符串设置请求body
func (r *Request) SetStringBody(s string) {
	r.SetReaderBody(strings.NewReader(s))
}

// ResetBody 回滚请求的body到开始的位置， 如果请求在被发送之前，
// body被读取了一部分，发送之前需要回滚body到开始位置
//
// 在请求发送到服务端之前， Build Handler会主动调用该方法设置请求body
func (r *Request) ResetBody() {
	body, err := r.getNextRequestBody()
	if err != nil {
		r.Error = err
		return
	}

	r.HTTPRequest.Body = body
}

// SetReaderBody 设置请求的body
func (r *Request) SetReaderBody(reader io.ReadSeeker) {
	r.Body = reader
	r.BodyStart, _ = reader.Seek(0, io.SeekCurrent) // Get the Bodies current offset.
	r.ResetBody()
}

const (
	willRetry   = "will retry"
	notRetrying = "not retrying"
	retryCount  = "retry %v/%v"
)

func debugLogReqError(r *Request, stage, retryStr string, err error) {
	if !r.Config.LogLevel.Matches(qiniu.LogDebugWithRequestErrors) {
		return
	}

	r.Config.Logger.Log(fmt.Sprintf("DEBUG: %s %s/%s failed, %s, error %v",
		stage, r.ServiceName, r.Api.Name(), retryStr, err))
}

// Build 构建请求，然后请求才可以被签名，发送到服务端
// 构建的过程中会对请求参数进行合法性检查
//
// 请求只可以构建一次， 多次调用Build只有第一次生效
//
// 如果构建过程中发生了错误，返回错误
func (r *Request) Build() error {
	if !r.built {
		r.Handlers.Validate.Run(r)
		if r.Error != nil {
			debugLogReqError(r, "Validate Request", notRetrying, r.Error)
			return r.Error
		}
		r.Handlers.Build.Run(r)
		if r.Error != nil {
			debugLogReqError(r, "Build Request", notRetrying, r.Error)
			return r.Error
		}
		r.built = true
	}

	return r.Error
}

// Sign 给发出的请求签名，如果遇到错误，直接返回错误信息
// 在签名之前会对请求进行构建
func (r *Request) Sign() error {
	r.Build()
	if r.Error != nil {
		debugLogReqError(r, "Build Request", notRetrying, r.Error)
		return r.Error
	}

	r.Handlers.Sign.Run(r)
	return r.Error
}

func (r *Request) getNextRequestBody() (io.ReadCloser, error) {
	if r.safeBody != nil {
		r.safeBody.Close()
	}

	r.safeBody = newOffsetReader(r.Body, r.BodyStart)

	l, err := qiniu.SeekerLen(r.Body)
	if err != nil {
		return nil, qerr.New(ErrCodeSerialization, "failed to compute request body size", err)
	}

	var body io.ReadCloser
	if l == 0 {
		body = http.NoBody
	} else if l > 0 {
		body = r.safeBody
	} else {
		// 阻止Transfer-Encoding: chunked的时候， GET, HEAD, DELETE方法带请求体
		switch r.HTTPRequest.Method {
		case "GET", "HEAD", "DELETE":
			body = http.NoBody
		default:
			body = r.safeBody
		}
	}

	return body, nil
}

// GetBody 返回一个io.ReadSeeker对象
func (r *Request) GetBody() io.ReadSeeker {
	return r.safeBody
}

// Send 发送请求到服务端， 如果遇到错误就返回错误信息
// 需求签名的请求会在发送之前进行签名，设置Authorization Header
// 所有的Handler会按照在Handler链表中的顺序进行调用
func (r *Request) Send() error {
	defer func() {
		r.Handlers.Complete.Run(r)
	}()

	if err := r.Error; err != nil {
		return err
	}

	for {
		r.Error = nil
		r.AttemptTime = time.Now()

		if err := r.Sign(); err != nil {
			debugLogReqError(r, "Sign Request", notRetrying, err)
			return err
		}

		if err := r.sendRequest(); err == nil {
			return nil
		} else if !shouldRetryCancel(r.Error) {
			return err
		} else {
			r.Handlers.Retry.Run(r)
			r.Handlers.AfterRetry.Run(r)

			if r.Error != nil || !qiniu.BoolValue(r.Retryable) {
				return r.Error
			}

			r.prepareRetry()
			continue
		}
	}
}

func (r *Request) prepareRetry() {
	if r.Config.LogLevel.Matches(qiniu.LogDebugWithRequestRetries) {
		r.Config.Logger.Log(fmt.Sprintf("DEBUG: Retrying Request %s/%s, attempt %d",
			r.Api.Name(), r.ServiceName, r.RetryCount))
	}

	r.HTTPRequest = copyHTTPRequest(r.HTTPRequest, nil)
	r.ResetBody()

	// 关闭response body， 防止重试的时候有网络链接泄漏
	if r.HTTPResponse != nil && r.HTTPResponse.Body != nil {
		r.HTTPResponse.Body.Close()
	}
}

func (r *Request) sendRequest() (sendErr error) {
	defer r.Handlers.CompleteAttempt.Run(r)

	r.Retryable = nil
	r.Handlers.Send.Run(r)
	if r.Error != nil {
		debugLogReqError(r, "Send Request",
			fmtAttemptCount(r.RetryCount, r.MaxRetries()),
			r.Error)
		return r.Error
	}

	r.Handlers.UnmarshalMeta.Run(r)
	r.Handlers.ValidateResponse.Run(r)
	if r.Error != nil {
		r.Handlers.UnmarshalError.Run(r)
		debugLogReqError(r, "Validate Response",
			fmtAttemptCount(r.RetryCount, r.MaxRetries()),
			r.Error)
		return r.Error
	}

	r.Handlers.Unmarshal.Run(r)
	if r.Error != nil {
		debugLogReqError(r, "Unmarshal Response",
			fmtAttemptCount(r.RetryCount, r.MaxRetries()),
			r.Error)
		return r.Error
	}

	return nil
}

func (r *Request) copy() *Request {
	req := &Request{}
	*req = *r
	req.Handlers = r.Handlers.Copy()
	return req
}

// AddToUserAgent 把字符串s加入到当前的UserAgent中
func AddToUserAgent(r *Request, s string) {
	curUA := r.HTTPRequest.Header.Get("User-Agent")
	if len(curUA) > 0 {
		s = curUA + " " + s
	}
	r.HTTPRequest.Header.Set("User-Agent", s)
}

type temporary interface {
	Temporary() bool
}

func shouldRetryCancel(origErr error) bool {
	switch err := origErr.(type) {
	case qerr.Error:
		if err.Code() == ErrCodeCanceled {
			return false
		}
		return shouldRetryCancel(err.OrigErr())
	case *url.Error:
		return shouldRetryCancel(err.Err)
	case temporary:
		// 如果是暂时性地错误， 我们希望重试请求
		return err.Temporary() || isErrConnectionReset(origErr)
	case nil:
		// `qerr.Error.OrigErr()` 可能是nil, 表示发生了错误， 但是不知道具体的原因, 我们认为这个是可以重试的
		return true
	default:
		switch err.Error() {
		case "net/http: request canceled",
			"net/http: request canceled while waiting for connection":
			return false
		}
		// 不知道什么错误， 默认允许重试
		return true
	}
}

// SanitizeHostForHeader 把默认端口号删除
func SanitizeHostForHeader(r *http.Request) {
	host := getHost(r)
	port := portOnly(host)
	if port != "" && isDefaultPort(r.URL.Scheme, port) {
		r.Host = stripPort(host)
	}
}

func getHost(r *http.Request) string {
	if r.Host != "" {
		return r.Host
	}

	return r.URL.Host
}

func stripPort(hostport string) string {
	colon := strings.IndexByte(hostport, ':')
	if colon == -1 {
		return hostport
	}
	if i := strings.IndexByte(hostport, ']'); i != -1 {
		return strings.TrimPrefix(hostport[:i], "[")
	}
	return hostport[:colon]
}

func portOnly(hostport string) string {
	colon := strings.IndexByte(hostport, ':')
	if colon == -1 {
		return ""
	}
	if i := strings.Index(hostport, "]:"); i != -1 {
		return hostport[i+len("]:"):]
	}
	if strings.Contains(hostport, "]") {
		return ""
	}
	return hostport[colon+len(":"):]
}

func isDefaultPort(scheme, port string) bool {
	if port == "" {
		return true
	}

	lowerCaseScheme := strings.ToLower(scheme)
	if (lowerCaseScheme == "http" && port == "80") || (lowerCaseScheme == "https" && port == "443") {
		return true
	}

	return false
}
