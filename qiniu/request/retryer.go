package request

import (
	"time"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/qerr"
)

// Retryer 接口控制请求重试的逻辑
// 默认的请求重试逻辑在client.DefaultRetryer中实现
type Retryer interface {
	// 防止和request.RetryDelay
	RetryRules(*Request) time.Duration
	ShouldRetry(*Request) bool
	MaxRetries() int
}

// WithRetryer 配置qiniu.Config中的重试逻辑
func WithRetryer(cfg *qiniu.Config, retryer Retryer) *qiniu.Config {
	cfg.Retryer = retryer
	return cfg
}

// retryableCodes 包含了可以重试的错误码
var retryableCodes = map[string]struct{}{
	qerr.ErrCrc32Verification: {},
	"RequestError":            {},
	"RequestTimeout":          {},
	ErrCodeResponseTimeout:    {},
	"RequestTimeoutException": {}, // Glacier's flavor of RequestTimeout
}

func isCodeRetryable(code string) bool {
	if _, ok := retryableCodes[code]; ok {
		return true
	}
	return false
}

var validParentCodes = map[string]struct{}{
	ErrCodeSerialization: {},
	ErrCodeRead:          {},
}

type temporaryError interface {
	Temporary() bool
}

func isNestedErrorRetryable(parentErr qerr.Error) bool {
	if parentErr == nil {
		return false
	}

	if _, ok := validParentCodes[parentErr.Code()]; !ok {
		return false
	}

	err := parentErr.OrigErr()
	if err == nil {
		return false
	}

	if aerr, ok := err.(qerr.Error); ok {
		return isCodeRetryable(aerr.Code())
	}

	if t, ok := err.(temporaryError); ok {
		return t.Temporary() || isErrConnectionReset(err)
	}

	return isErrConnectionReset(err)
}

// IsErrorRetryable 判断错误是否可以重试
func IsErrorRetryable(err error) bool {
	if err != nil {
		if aerr, ok := err.(qerr.Error); ok {
			return isCodeRetryable(aerr.Code()) || isNestedErrorRetryable(aerr)
		}
	}
	return false
}

// IsErrorRetryable 返回true， 如果错误可以重试， 否则false
// 根据请求错误的错误码来判断错误是否可以重试
func (r *Request) IsErrorRetryable() bool {
	return IsErrorRetryable(r.Error)
}
