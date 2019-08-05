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

// Interface for matching types which also have a Len method.
type lener interface {
	Len() int
}

// BuildContentLengthHandler builds the content length of a request based on the body,
// or will use the HTTPRequest.Header's "Content-Length" if defined. If unable
// to determine request body length and no "Content-Length" was specified it will panic.
//
// The Content-Length will only be added to the request if the length of the body
// is greater than 0. If the body is empty or the current `Content-Length`
// header is <= 0, the header will also be stripped.
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

// SendHandler is a request handler to send service request using HTTP client.
var SendHandler = request.NamedHandler{
	Name: "core.SendHandler",
	Fn: func(r *request.Request) {
		sender := sendFollowRedirects
		if r.DisableFollowRedirects {
			sender = sendWithoutFollowRedirects
		}

		if http.NoBody == r.HTTPRequest.Body {
			// Strip off the request body if the NoBody reader was used as a
			// place holder for a request body. This prevents the SDK from
			// making requests with a request body when it would be invalid
			// to do so.
			//
			// Use a shallow copy of the http.Request to ensure the race condition
			// of transport on Body will not trigger
			reqOrig, reqCopy := r.HTTPRequest, *r.HTTPRequest
			reqCopy.Body = nil
			r.HTTPRequest = &reqCopy
			defer func() {
				r.HTTPRequest = reqOrig
			}()
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
	// Prevent leaking if an HTTPResponse was returned. Clean up
	// the body.
	if r.HTTPResponse != nil {
		r.HTTPResponse.Body.Close()
	}
	// Capture the case where url.Error is returned for error processing
	// response. e.g. 301 without location header comes back as string
	// error and r.HTTPResponse is nil. Other URL redirect errors will
	// comeback in a similar method.
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
		// Add a dummy request response object to ensure the HTTPResponse
		// value is consistent.
		r.HTTPResponse = &http.Response{
			StatusCode: int(0),
			Status:     http.StatusText(int(0)),
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
		}
	}
	// Catch all other request errors.
	r.Error = qerr.New("RequestError", "send request failed", err)
	r.Retryable = qiniu.Bool(true) // network errors are retryable

	// Override the error with a context canceled error, if that was canceled.
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

// ValidateResponseHandler is a request handler to validate service response.
var ValidateResponseHandler = request.NamedHandler{Name: "core.ValidateResponseHandler", Fn: func(r *request.Request) {
	if r.HTTPResponse.StatusCode == 0 || r.HTTPResponse.StatusCode >= 300 {
		var errMsg string
		contentLength := r.HTTPResponse.Header.Get("Content-Length")

		length, pErr := strconv.ParseInt(contentLength, 10, 64)
		if pErr != nil {
			r.Error = qerr.New(qerr.ConvertError, fmt.Sprintf("convert string `%s` to int error", contentLength), pErr)
		}
		if length > 0 {
			if t := r.HTTPResponse.Header.Get("Content-Type"); t == "application/json" {
				var em ErrMsg
				var bf bytes.Buffer

				defer r.HTTPResponse.Body.Close()

				err := json.NewDecoder(io.TeeReader(r.HTTPResponse.Body, &bf)).Decode(&em)
				if err != nil {
					r.Error = qerr.New(qerr.DecodeError, "decode json data error", err)
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
		case 401:
			r.Error = qerr.New(qerr.AuthorizationError, errMsg, nil)
		case 400:
			r.Error = qerr.New(qerr.ParamsError, errMsg, nil)
		default:
			r.Error = qerr.New("UnknownError", errMsg, nil)
		}
	}
}}

// AfterRetryHandler performs final checks to determine if the request should
// be retried and how long to delay.
var AfterRetryHandler = request.NamedHandler{Name: "core.AfterRetryHandler", Fn: func(r *request.Request) {
	// If one of the other handlers already set the retry state
	// we don't want to override it based on the service's state
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
