package corehandlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/qiniu/go-sdk/internal/encoding"
	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/qerr"
	"github.com/qiniu/go-sdk/qiniu/request"
)

type lener interface {
	Len() int
}

// BuildContentLengthHandler 计算请求体的长度
// 首先使用request.Body计算请求体的长度， 如果计算失败并且"Content-Length"设置了相应的长度， 那么使用该长度
// 如果计算request.Body失败，Content-Length也没有设置，那么返回错误信息
//
// 只有当Content-Length 的值大于0的时候，才会设置请求头， 否则发出的请求中不包含该请求头
var BuildContentLengthHandler = request.NamedHandler{Name: "core.BuildContentLengthHandler", Fn: func(r *request.Request) {
	var length int64

	if slength := r.HTTPRequest.Header.Get("Content-Length"); slength != "" {
		length, _ = strconv.ParseInt(slength, 10, 64)
	} else {
		if r.Body != nil {
			var err error
			length, err = qiniu.SeekerLen(r.Body)
			if err != nil {
				r.Error = qerr.New(request.ErrCodeSerialization, "failed to get request body's length", err)
				return
			}
		}
	}

	if length > 0 {
		r.HTTPRequest.ContentLength = length
		r.HTTPRequest.Header.Set("Content-Length", fmt.Sprintf("%d", length))
	} else {
		r.HTTPRequest.ContentLength = 0
		r.HTTPRequest.Header.Del("Content-Length")
	}
}}

var reStatusCode = regexp.MustCompile(`^(\d{3})`)

// SendHandler 发出http 请求
var SendHandler = request.NamedHandler{
	Name: "core.SendHandler",
	Fn: func(r *request.Request) {
		sender := sendFollowRedirects
		if r.DisableFollowRedirects {
			sender = sendWithoutFollowRedirects
		}
		var err error
		r.HTTPResponse, err = sender(r)
		if err != nil {
			handleSendError(r, err)
		}
	},
}

// BodyHandler 根据输入的类型r.Params和request.Content-Type来选择合适的Encoder
// 序列化结构体到http 请求体中
var BodyHandler = request.NamedHandler{
	Name: "core.BodyHandler",
	Fn: func(r *request.Request) {
		if r.ParamsFilled() {
			switch r.HTTPRequest.Header.Get("Content-Type") {
			case "application/json":
				data, err := json.Marshal(r.Params)
				if err != nil {
					r.Error = qerr.New(request.ErrCodeSerialization, "failed to encode application/json data", err)
					return
				}
				r.SetBufferBody(data)
			case "application/x-www-form-urlencoded":
				v := make(url.Values)
				err := encoding.NewEncoder().Encode(r.Params, v)
				if err != nil {
					r.Error = qerr.New(request.ErrCodeSerialization, "failed to encode application/x-www-form-urlencoded data", err)
					return
				}
				r.SetStringBody(v.Encode())
			default: // application/octet-stream, etc
				if reader, ok := r.Params.(io.ReadSeeker); ok {
					r.SetReaderBody(reader)
				} else if bs, ok := r.Params.(*[]byte); ok {
					r.SetBufferBody(*bs)
				} else {
					r.Error = qerr.New(request.ErrCodeSerialization, "request Params must be io.ReadSeeker for content-type: "+r.HTTPRequest.Header.Get("Content-Type"), nil)
					return
				}
			}
		}
	},
}

func sendFollowRedirects(r *request.Request) (*http.Response, error) {
	return r.Config.HTTPClient.Do(r.HTTPRequest)
}

func sendWithoutFollowRedirects(r *request.Request) (*http.Response, error) {
	transport := r.Config.HTTPClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	return transport.RoundTrip(r.HTTPRequest)
}

func handleSendError(r *request.Request, err error) {
	// 关闭Response.Body, 防止泄漏
	if r.HTTPResponse != nil {
		r.HTTPResponse.Body.Close()
	}
	// 捕获url.Error， 比如301跳转响应没有设置Location Header
	if e, ok := err.(*url.Error); ok && e.Err != nil {
		if s := reStatusCode.FindStringSubmatch(e.Err.Error()); s != nil {
			code, _ := strconv.ParseInt(s[1], 10, 64)
			r.HTTPResponse = &http.Response{
				StatusCode: int(code),
				Status:     http.StatusText(int(code)),
				Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
			}
			return
		}
	}
	if r.HTTPResponse == nil {
		// 为了使r.HTTPResponse保持一致， 不为nil
		r.HTTPResponse = &http.Response{
			StatusCode: int(0),
			Status:     http.StatusText(int(0)),
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
		}
	}
	r.Error = qerr.New("RequestError", "send request failed", err)
	r.Retryable = qiniu.Bool(true) // network errors are retryable

	// 如果是请求是被取消了， 设置r.Error为被取消
	ctx := r.Context()
	select {
	case <-ctx.Done():
		r.Error = qerr.New(request.CanceledErrorCode,
			"request context canceled", ctx.Err())
		r.Retryable = qiniu.Bool(false)
	default:
	}
}

// ErrMsg 结构体用于保存接口返回的application/json错误信息
type ErrMsg struct {
	Err string `json:"error"`
}

// ValidateResponseHandler 校验响应信息， 设置响应的错误信息
var ValidateResponseHandler = request.NamedHandler{Name: "core.ValidateResponseHandler", Fn: func(r *request.Request) {
	if r.HTTPResponse.StatusCode == 0 || r.HTTPResponse.StatusCode >= 300 {
		var errMsg string
		contentLength := r.HTTPResponse.Header.Get("Content-Length")

		length, pErr := strconv.ParseInt(contentLength, 10, 64)
		if pErr != nil {
			r.Error = qerr.New(qerr.ErrConvertTypes, fmt.Sprintf("convert string `%s` to int error", contentLength), pErr)
		}
		if length > 0 {
			if t := r.HTTPResponse.Header.Get("Content-Type"); t == "application/json" {
				var em ErrMsg
				var bf bytes.Buffer

				defer r.HTTPResponse.Body.Close()

				err := json.NewDecoder(io.TeeReader(r.HTTPResponse.Body, &bf)).Decode(&em)
				if err != nil {
					r.Error = qerr.New(qerr.ErrCodeDeserialization, "decode json data error", err)
				}
				if err == nil && em.Err != "" {
					errMsg = em.Err
				}

				r.HTTPResponse.Body = ioutil.NopCloser(bytes.NewReader(bf.Bytes()))
			}
		}
		if errMsg != "" {
			errMsg = r.HTTPResponse.Status + ": " + errMsg
		} else {
			errMsg = r.HTTPResponse.Status
		}
		// this may be replaced by an UnmarshalError handler
		switch r.HTTPResponse.StatusCode {
		case 298:
			r.Error = qerr.New(qerr.ErrPartFailed, errMsg, nil)
		case 400:
			r.Error = qerr.New(qerr.ErrParams, errMsg, nil)
		case 401:
			r.Error = qerr.New(qerr.ErrAuthorization, errMsg, nil)
		case 403:
			r.Error = qerr.New(qerr.ErrAccessForbidden, errMsg, nil)
		case 404:
			r.Error = qerr.New(qerr.ErrNotFound, errMsg, nil)
		case 405:
			r.Error = qerr.New(qerr.ErrUnexpectedRequest, errMsg, nil)
		case 406:
			r.Error = qerr.New(qerr.ErrCrc32Verification, errMsg, nil)
		case 419:
			r.Error = qerr.New(qerr.ErrAccountFrozen, errMsg, nil)
		case 478:
			r.Error = qerr.New(qerr.ErrMirrorSourceRequest, errMsg, nil)
		case 503:
			r.Error = qerr.New(qerr.ErrServiceUnavailable, errMsg, nil)
		case 504:
			r.Error = qerr.New(qerr.ErrServiceTimeout, errMsg, nil)
		case 573:
			r.Error = qerr.New(qerr.ErrRequestRate, errMsg, nil)
		case 579:
			r.Error = qerr.New(qerr.ErrUploadCallback, errMsg, nil)
		case 599:
			r.Error = qerr.New(qerr.ErrServiceOps, errMsg, nil)
		case 608:
			r.Error = qerr.New(qerr.ErrContentChanged, errMsg, nil)
		case 612:
			r.Error = qerr.New(qerr.ErrResourceNotExist, errMsg, nil)
		case 614:
			r.Error = qerr.New(qerr.ErrResourceExist, errMsg, nil)
		default:
			r.Error = qerr.New(qerr.ErrUnknown, errMsg, nil)
		}
	}
}}

// AfterRetryHandler 决定请求是否重试，重试的间隔多长
var AfterRetryHandler = request.NamedHandler{Name: "core.AfterRetryHandler", Fn: func(r *request.Request) {
	if r.Retryable == nil || qiniu.BoolValue(r.Config.EnforceShouldRetryCheck) {
		r.Retryable = qiniu.Bool(r.ShouldRetry(r))
	}

	if r.WillRetry() {
		r.RetryDelay = r.RetryRules(r)

		if err := qiniu.SleepWithContext(r.Context(), r.RetryDelay); err != nil {
			r.Error = qerr.New(request.CanceledErrorCode,
				"request context canceled", err)
			r.Retryable = qiniu.Bool(false)
			return
		}

		r.RetryCount++
		r.Error = nil
	}
}}

// CompleteHandler 关闭response.Body
var CompleteHandler = request.NamedHandler{
	Name: "core.CompleteHandler",
	Fn: func(r *request.Request) {
		if r.HTTPResponse != nil && r.HTTPResponse.Body != nil {
			r.HTTPResponse.Body.Close()
		}
	},
}
