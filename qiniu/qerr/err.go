package qerr

import (
	"fmt"
)

// Error 以错误码， 错误信息， 原始错误归类错误
// SDK 中所有的错误信息都实现了该接口或者error接口
//
// 调用Error() 或者 String() 方法会返回完整的错误信息，包括封装的底层错误的信息
type Error interface {
	// 实现一般的错误接口
	error

	// 返回错误码
	Code() string

	// 返回具体的错误信息
	Message() string

	// 返回原始的错误，如果设置了的话， 不然返回nil
	OrigErr() error
}

// BatchedErrors 是一系列关联的错误， 调用Error()方法会返回所有的错误信息
type BatchedErrors interface {
	// 满足Error接口
	Error

	// 返回原始的一串错误， 如果没有设置，返回nil
	OrigErrs() []error
}

// New 返回一个Error对象
func New(code, message string, origErr error) Error {
	var errs []error
	if origErr != nil {
		errs = append(errs, origErr)
	}
	return newBaseError(code, message, errs)
}

// NewBatchedError 返回一个BatchedErrors对象
func NewBatchedError(code, message string, errs []error) BatchedErrors {
	return newBaseError(code, message, errs)
}

// RequestFailure 是请求错误接口， 封装了错误请求的ID， 服务段返回的状态码
// 错误请求可能没有requestID， 比如请求还没有达到服务端就遇到了网络错误
type RequestFailure interface {
	Error

	// HTTP 响应的状态码
	StatusCode() int

	// 服务端响应的错误请求状态码
	RequestID() string
}

// NewRequestFailure 返回 RequestFailure对象
func NewRequestFailure(err Error, statusCode int, reqID string) RequestFailure {
	return newRequestError(err, statusCode, reqID)
}

// SprintError 返回格式化的错误信息
//
// 其中extra, origErr 是可选的参数.  如果设置了这两个参数，那么错误信息会加入相应的信息，
// 否则就没有
func SprintError(code, message, extra string, origErr error) string {
	msg := fmt.Sprintf("%s: %s", code, message)
	if extra != "" {
		msg = fmt.Sprintf("%s\n\t%s", msg, extra)
	}
	if origErr != nil {
		msg = fmt.Sprintf("%s\ncaused by: %s", msg, origErr.Error())
	}
	return msg
}

// baseError 封装了code, message字段， 定义了错误信息的类别码， 具体的错误信息
type baseError struct {
	// 错误类别码
	code string

	// 具体的错误信息
	message string

	// 引起该错误的原始的一系列错误
	errs []error
}

// newBaseErrror 返回baseError指针
func newBaseError(code, message string, origErrs []error) *baseError {
	b := &baseError{
		code:    code,
		message: message,
		errs:    origErrs,
	}

	return b
}

// Error 返回错误信息的字符串表示
// 实现error接口
func (b baseError) Error() string {
	size := len(b.errs)
	if size > 0 {
		return SprintError(b.code, b.message, "", errorList(b.errs))
	}

	return SprintError(b.code, b.message, "", nil)
}

// String 返回错误信息的字符串表示, 和Error方法返回相同
func (b baseError) String() string {
	return b.Error()
}

// Code 返回错误码
func (b baseError) Code() string {
	return b.code
}

// Message 返回具体的错误信息
func (b baseError) Message() string {
	return b.message
}

// OrigErr 如果设置了原始的错误， 返回原始的错误, 否则返回nil
// 如果有多个原始的错误， 只返回第一个
func (b baseError) OrigErr() error {
	switch len(b.errs) {
	case 0:
		return nil
	case 1:
		return b.errs[0]
	default:
		if err, ok := b.errs[0].(Error); ok {
			return NewBatchedError(err.Code(), err.Message(), b.errs[1:])
		}
		return NewBatchedError("BatchedErrors",
			"multiple errors occurred", b.errs)
	}
}

// OrigErrs 返回原始的多个错误， 如果没有设置，返回空的切片
func (b baseError) OrigErrs() []error {
	return b.errs
}

// 为了Error接口可以以匿名的字段设置在requestError中， 避免与error.Error() 方法冲突
type qiniuError Error

// requestError 代表请求错误
type requestError struct {
	qiniuError
	statusCode int
	requestID  string
	bytes      []byte
}

func newRequestError(err Error, statusCode int, requestID string) *requestError {
	return &requestError{
		qiniuError: err,
		statusCode: statusCode,
		requestID:  requestID,
	}
}

// Error returns the string representation of the error.
// Satisfies the error interface.
func (r requestError) Error() string {
	extra := fmt.Sprintf("status code: %d, request id: %s",
		r.statusCode, r.requestID)
	return SprintError(r.Code(), r.Message(), extra, r.OrigErr())
}

// String returns the string representation of the error.
// Alias for Error to satisfy the stringer interface.
func (r requestError) String() string {
	return r.Error()
}

// StatusCode returns the wrapped status code for the error
func (r requestError) StatusCode() int {
	return r.statusCode
}

// RequestID returns the wrapped requestID
func (r requestError) RequestID() string {
	return r.requestID
}

// OrigErrs returns the original errors if one was set. An empty slice is
// returned if no error was set.
func (r requestError) OrigErrs() []error {
	if b, ok := r.qiniuError.(BatchedErrors); ok {
		return b.OrigErrs()
	}
	return []error{r.OrigErr()}
}

// An error list that satisfies the golang interface
type errorList []error

// Error returns the string representation of the error.
//
// Satisfies the error interface.
func (e errorList) Error() string {
	msg := ""
	// How do we want to handle the array size being zero
	if size := len(e); size > 0 {
		for i := 0; i < size; i++ {
			msg += fmt.Sprintf("%s", e[i].Error())
			// We check the next index to see if it is within the slice.
			// If it is, then we append a newline. We do this, because unit tests
			// could be broken with the additional '\n'
			if i+1 < size {
				msg += "\n"
			}
		}
	}
	return msg
}
