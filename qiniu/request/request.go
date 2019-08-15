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
	// ErrCodeSerialization is the serialization error code that is received
	// during protocol marshaling.
	ErrCodeSerialization = "SerializationError"

	// ErrCodeRead is an error that is returned during HTTP reads.
	ErrCodeRead = "ReadError"

	// ErrCodeResponseTimeout is the connection timeout error that is received
	// during body reads.
	ErrCodeResponseTimeout = "ResponseTimeout"

	// CanceledErrorCode is the error code that will be returned by an
	// API request that was canceled. Requests given a context.Context may
	// return this error when canceled.
	CanceledErrorCode = "RequestCanceled"
)

// A Request is the service request to be made.
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
	BodyStart              int64 // offset from beginning of Body that the request body starts
	Params                 interface{}
	Error                  error
	Data                   interface{}
	RequestID              string
	RetryCount             int
	Retryable              *bool
	RetryDelay             time.Duration
	SignedHeaderVals       http.Header
	LastSignedAt           time.Time
	DisableFollowRedirects bool

	// A value greater than 0 instructs the request to be signed as Presigned URL
	// You should not set this field directly. Instead use Request's
	// Presign or PresignRequest methods.
	context context.Context

	built bool

	// Need to persist an intermediate body between the input Body and HTTP
	// request body because the HTTP Client's transport can maintain a reference
	// to the HTTP request's body after the client has returned. This value is
	// safe to use concurrently and wrap the input Body for each HTTP request.
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

func (a *API) Name() string {
	return a.APIName
}

func (a *API) url() string {
	var scheme string

	if a.Scheme == "" {
		scheme = "http"
	} else {
		scheme = a.Scheme
	}
	host := strings.TrimRight(a.Host, "/")
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
	httpReq.URL, err = url.Parse(operation.url())
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

// A Option is a functional option that can augment or modify a request when
// using a WithContext API operation method.
type Option func(*Request)

// WithGetResponseHeader builds a request Option which will retrieve a single
// header value from the HTTP Response. If there are multiple values for the
// header key use WithGetResponseHeaders instead to access the http.Header
// map directly. The passed in val pointer must be non-nil.
func WithGetResponseHeader(key string, val *string) Option {
	return func(r *Request) {
		r.Handlers.Complete.PushBack(func(req *Request) {
			*val = req.HTTPResponse.Header.Get(key)
		})
	}
}

// WithGetResponseHeaders builds a request Option which will retrieve the
// headers from the HTTP response and assign them to the passed in headers
// variable. The passed in headers pointer must be non-nil.
func WithGetResponseHeaders(headers *http.Header) Option {
	return func(r *Request) {
		r.Handlers.Complete.PushBack(func(req *Request) {
			*headers = req.HTTPResponse.Header
		})
	}
}

// WithLogLevel is a request option that will set the request to use a specific
// log level when the request is made.
func WithLogLevel(l qiniu.LogLevelType) Option {
	return func(r *Request) {
		r.Config.LogLevel = qiniu.LogLevel(l)
	}
}

// ApplyOptions will apply each option to the request calling them in the order
// the were provided.
func (r *Request) ApplyOptions(opts ...Option) {
	for _, opt := range opts {
		opt(r)
	}
}

// Context will always returns a non-nil context. If Request does not have a
// context context.BackgroundContext will be returned.
func (r *Request) Context() context.Context {
	if r.context != nil {
		return r.context
	}
	return context.Background()
}

// SetContext adds a Context to the current request that can be used to cancel
// a in-flight request. The Context value must not be nil, or this method will
// panic.
//
// Unlike http.Request.WithContext, SetContext does not return a copy of the
// Request. It is not safe to use use a single Request value for multiple
// requests. A new Request should be created for each API operation request.
func (r *Request) SetContext(ctx context.Context) {
	if ctx == nil {
		panic("context cannot be nil")
	}
	r.context = ctx
	r.HTTPRequest = r.HTTPRequest.WithContext(ctx)
}

// WillRetry returns if the request's can be retried.
func (r *Request) WillRetry() bool {
	if !qiniu.IsReaderSeekable(r.Body) && r.HTTPRequest.Body != http.NoBody {
		return false
	}
	return r.Error != nil && qiniu.BoolValue(r.Retryable) && r.RetryCount < r.MaxRetries()
}

func fmtAttemptCount(retryCount, maxRetries int) string {
	return fmt.Sprintf("attempt %v/%v", retryCount, maxRetries)
}

// ParamsFilled returns if the request's parameters have been populated
// and the parameters are valid. False is returned if no parameters are
// provided or invalid.
func (r *Request) ParamsFilled() bool {
	return r.Params != nil && reflect.ValueOf(r.Params).Elem().IsValid()
}

// DataFilled returns true if the request's data for response deserialization
// target has been set and is a valid. False is returned if data is not
// set, or is invalid.
func (r *Request) DataFilled() bool {
	return r.Data != nil && reflect.ValueOf(r.Data).Elem().IsValid()
}

// SetBufferBody will set the request's body bytes that will be sent to
// the service API.
func (r *Request) SetBufferBody(buf []byte) {
	r.SetReaderBody(bytes.NewReader(buf))
}

// SetStringBody sets the body of the request to be backed by a string.
func (r *Request) SetStringBody(s string) {
	r.SetReaderBody(strings.NewReader(s))
}

// ResetBody rewinds the request body back to its starting position, and
// sets the HTTP Request body reference. When the body is read prior
// to being sent in the HTTP request it will need to be rewound.
//
// ResetBody will automatically be called by the SDK's build handler, but if
// the request is being used directly ResetBody must be called before the request
// is Sent.  SetStringBody, SetBufferBody, and SetReaderBody will automatically
// call ResetBody.
func (r *Request) ResetBody() {
	body, err := r.getNextRequestBody()
	if err != nil {
		r.Error = err
		return
	}

	r.HTTPRequest.Body = body
}

// SetReaderBody will set the request's body reader.
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

// Build will build the request's object so it can be signed and sent
// to the service. Build will also validate all the request's parameters.
// Any additional build Handlers set on this request will be run
// in the order they were set.
//
// The request will only be built once. Multiple calls to build will have
// no effect.
//
// If any Validate or Build errors occur the build will stop and the error
// which occurred will be returned.
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

// Sign will sign the request, returning error if errors are encountered.
//
// Sign will build the request prior to signing. All Sign Handlers will
// be executed in the order they were set.
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
		// Hack to prevent sending bodies for methods where the body
		// should be ignored by the server. Sending bodies on these
		// methods without an associated ContentLength will cause the
		// request to socket timeout because the server does not handle
		// Transfer-Encoding: chunked bodies for these methods.
		//
		// This would only happen if a api.ReaderSeekerCloser was used with
		// a io.Reader that was not also an io.Seeker, or did not implement
		// Len() method.
		switch r.HTTPRequest.Method {
		case "GET", "HEAD", "DELETE":
			body = http.NoBody
		default:
			body = r.safeBody
		}
	}

	return body, nil
}

// GetBody will return an io.ReadSeeker of the Request's underlying
// input body with a concurrency safe wrapper.
func (r *Request) GetBody() io.ReadSeeker {
	return r.safeBody
}

// Send will send the request, returning error if errors are encountered.
//
// Send will sign the request prior to sending. All Send Handlers will
// be executed in the order they were set.
//
// Canceling a request is non-deterministic. If a request has been canceled,
// then the transport will choose, randomly, one of the state channels during
// reads or getting the connection.
//
// readLoop() and getConn(req *Request, cm connectMethod)
// https://github.com/golang/go/blob/master/src/net/http/transport.go
//
// Send will not close the request.Request's body.
func (r *Request) Send() error {
	defer func() {
		// Regardless of success or failure of the request trigger the Complete
		// request handlers.
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

	// The previous http.Request will have a reference to the r.Body
	// and the HTTP Client's Transport may still be reading from
	// the request's body even though the Client's Do returned.
	r.HTTPRequest = copyHTTPRequest(r.HTTPRequest, nil)
	r.ResetBody()

	// Closing response body to ensure that no response body is leaked
	// between retry attempts.
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

// copy will copy a request which will allow for local manipulation of the
// request.
func (r *Request) copy() *Request {
	req := &Request{}
	*req = *r
	req.Handlers = r.Handlers.Copy()
	return req
}

// AddToUserAgent adds the string to the end of the request's current user agent.
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
		if err.Code() == CanceledErrorCode {
			return false
		}
		return shouldRetryCancel(err.OrigErr())
	case *url.Error:
		if strings.Contains(err.Error(), "connection refused") {
			// Refused connections should be retried as the service may not yet
			// be running on the port. Go TCP dial considers refused
			// connections as not temporary.
			return true
		}
		// *url.Error only implements Temporary after golang 1.6 but since
		// url.Error only wraps the error:
		return shouldRetryCancel(err.Err)
	case temporary:
		// If the error is temporary, we want to allow continuation of the
		// retry process
		return err.Temporary() || isErrConnectionReset(origErr)
	case nil:
		// `qerr.Error.OrigErr()` can be nil, meaning there was an error but
		// because we don't know the cause, it is marked as retryable. See
		// TestRequest4xxUnretryable for an example.
		return true
	default:
		switch err.Error() {
		case "net/http: request canceled",
			"net/http: request canceled while waiting for connection":
			// known 1.5 error case when an http request is cancelled
			return false
		}
		// here we don't know the error; so we allow a retry.
		return true
	}
}

// SanitizeHostForHeader removes default port from host and updates request.Host
func SanitizeHostForHeader(r *http.Request) {
	host := getHost(r)
	port := portOnly(host)
	if port != "" && isDefaultPort(r.URL.Scheme, port) {
		r.Host = stripPort(host)
	}
}

// Returns host from request
func getHost(r *http.Request) string {
	if r.Host != "" {
		return r.Host
	}

	return r.URL.Host
}

// Hostname returns u.Host, without any port number.
//
// If Host is an IPv6 literal with a port number, Hostname returns the
// IPv6 literal without the square brackets. IPv6 literals may include
// a zone identifier.
//
// Copied from the Go 1.8 standard library (net/url)
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

// Port returns the port part of u.Host, without the leading colon.
// If u.Host doesn't contain a port, Port returns an empty string.
//
// Copied from the Go 1.8 standard library (net/url)
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

// Returns true if the specified URI is using the standard port
// (i.e. port 80 for HTTP URIs or 443 for HTTPS URIs)
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
