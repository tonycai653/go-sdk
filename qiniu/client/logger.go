package client

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httputil"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/request"
)

const logReqMsg = `DEBUG: Request %s/%s Details:
---[ REQUEST POST-SIGN ]-----------------------------
%s
-----------------------------------------------------`

const logReqErrMsg = `DEBUG ERROR: Request %s/%s:
---[ REQUEST DUMP ERROR ]-----------------------------
%s
------------------------------------------------------`

type logWriter struct {
	Logger qiniu.Logger
	buf    *bytes.Buffer
}

func (logger *logWriter) Write(b []byte) (int, error) {
	return logger.buf.Write(b)
}

type teeReaderCloser struct {
	// io.Reader 的值将会为tee reader, 用来从source读取内容，并且内容读取的内容写入到logger中
	io.Reader

	// Source数据来源
	Source io.ReadCloser
}

func (reader *teeReaderCloser) Close() error {
	return reader.Source.Close()
}

// LogHTTPRequestHandler 输出请求日志
// 当LogLevel满足LogDebugWithHTTPBody的时候， 也输出请求体的日志
var LogHTTPRequestHandler = request.NamedHandler{
	Name: "qiniusdk.client.LogRequest",
	Fn:   logRequest,
}

func logRequest(r *request.Request) {
	logBody := r.Config.LogLevel.Matches(qiniu.LogDebugWithHTTPBody)
	bodySeekable := qiniu.IsReaderSeekable(r.Body)

	b, err := httputil.DumpRequestOut(r.HTTPRequest, logBody)
	if err != nil {
		r.Config.Logger.Log(fmt.Sprintf(logReqErrMsg,
			r.ServiceName, r.Api.Name(), err))
		return
	}

	if logBody {
		if !bodySeekable {
			r.SetReaderBody(qiniu.ReadSeekCloser(r.HTTPRequest.Body))
		}
		r.ResetBody()
	}

	r.Config.Logger.Log(fmt.Sprintf(logReqMsg,
		r.ServiceName, r.Api.Name(), string(b)))
}

// LogHTTPRequestHeaderHandler 仅打印输出请求头的日志
var LogHTTPRequestHeaderHandler = request.NamedHandler{
	Name: "qiniusdk.client.LogRequestHeader",
	Fn:   logRequestHeader,
}

func logRequestHeader(r *request.Request) {
	b, err := httputil.DumpRequestOut(r.HTTPRequest, false)
	if err != nil {
		r.Config.Logger.Log(fmt.Sprintf(logReqErrMsg,
			r.ServiceName, r.Api.Name(), err))
		return
	}

	r.Config.Logger.Log(fmt.Sprintf(logReqMsg,
		r.ServiceName, r.Api.Name(), string(b)))
}

const logRespMsg = `DEBUG: Response %s/%s Details:
---[ RESPONSE ]--------------------------------------
%s
-----------------------------------------------------`

const logRespErrMsg = `DEBUG ERROR: Response %s/%s:
---[ RESPONSE DUMP ERROR ]-----------------------------
%s
-----------------------------------------------------`

// LogHTTPResponseHandler 输出响应的日志
// 当LogLevel满足LogDebugWithHTTPBody的, 也输出响应体
var LogHTTPResponseHandler = request.NamedHandler{
	Name: "qiniusdk.client.LogResponse",
	Fn:   logResponse,
}

func logResponse(r *request.Request) {
	lw := &logWriter{r.Config.Logger, bytes.NewBuffer(nil)}

	if r.HTTPResponse == nil {
		lw.Logger.Log(fmt.Sprintf(logRespErrMsg,
			r.ServiceName, r.Api.Name(), "request's HTTPResponse is nil"))
		return
	}

	logBody := r.Config.LogLevel.Matches(qiniu.LogDebugWithHTTPBody)
	if logBody {
		r.HTTPResponse.Body = &teeReaderCloser{
			Reader: io.TeeReader(r.HTTPResponse.Body, lw),
			Source: r.HTTPResponse.Body,
		}
	}

	handlerFn := func(req *request.Request) {
		b, err := httputil.DumpResponse(req.HTTPResponse, false)
		if err != nil {
			lw.Logger.Log(fmt.Sprintf(logRespErrMsg,
				req.ServiceName, req.Api.Name(), err))
			return
		}
		lw.Logger.Log(fmt.Sprintf(logRespMsg,
			req.ServiceName, req.Api.Name(), string(b)))

		if logBody {
			b, err := ioutil.ReadAll(lw.buf)
			if err != nil {
				lw.Logger.Log(fmt.Sprintf(logRespErrMsg,
					req.ServiceName, req.Api.Name(), err))
				return
			}

			lw.Logger.Log(string(b))
		}
	}

	const handlerName = "qiniusdk.client.LogResponse.ResponseBody"

	r.Handlers.Unmarshal.PushFrontNamed(request.NamedHandler{
		Name: handlerName, Fn: handlerFn,
	})
	r.Handlers.UnmarshalError.SetBackNamed(request.NamedHandler{
		Name: handlerName, Fn: handlerFn,
	})
}

// LogHTTPResponseHeaderHandler 输出响应信息日志，仅输出响应头信息
var LogHTTPResponseHeaderHandler = request.NamedHandler{
	Name: "qiniusdk.client.LogResponseHeader",
	Fn:   logResponseHeader,
}

func logResponseHeader(r *request.Request) {
	if r.Config.Logger == nil {
		return
	}

	b, err := httputil.DumpResponse(r.HTTPResponse, false)
	if err != nil {
		r.Config.Logger.Log(fmt.Sprintf(logRespErrMsg,
			r.ServiceName, r.Api.Name(), err))
		return
	}

	r.Config.Logger.Log(fmt.Sprintf(logRespMsg,
		r.ServiceName, r.Api.Name(), string(b)))
}
