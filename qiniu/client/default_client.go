package client

import (
	"strconv"
	"time"

	"github.com/qiniu/go-sdk/qiniu/request"
)

// DefaultRetryer 实现了请求重试的默认逻辑
// 如果想自己实现请求重试的逻辑， 可以实现request.Retryer接口
// 或者把DefaultRetryer内嵌在结构体中，然后重写相应的方法, 比如重写MaxRetries()方法
//
//	  type retryer struct {
//        client.DefaultRetryer
//    }
//
//    // 这个实现最多可以重试100次请求
//    func (d retryer) MaxRetries() int { return 100 }
type DefaultRetryer struct {
	NumMaxRetries int
}

// MaxRetries 返回最大的重试次数
func (d DefaultRetryer) MaxRetries() int {
	return d.NumMaxRetries
}

// RetryRules 返回重试请求之前的时间间隔, 会遵循After-Retry的值
// 如果没有该请求头， 默认3s
func (d DefaultRetryer) RetryRules(r *request.Request) time.Duration {
	if delay, retry := getRetryDelay(r); retry {
		return delay
	}
	return 3 * time.Second
}

// ShouldRetry 判断请求是否可以重试
func (d DefaultRetryer) ShouldRetry(r *request.Request) bool {
	if r.Retryable != nil {
		return *r.Retryable
	}
	// 501 - functionality not supported
	// 429 - too many requests
	// 503 - service unavailable
	// 406 - 上传校验失败
	switch r.HTTPResponse.StatusCode {
	case 501:
	case 429:
	case 503:
		return false
	case 406:
		return true
	}

	return r.IsErrorRetryable()
}

// 根据Retry-After 头判断重试的间隔, RFC 7231
func getRetryDelay(r *request.Request) (time.Duration, bool) {

	delayStr := r.HTTPResponse.Header.Get("Retry-After")
	if len(delayStr) == 0 {
		return 0, false
	}

	delay, err := strconv.Atoi(delayStr)
	if err != nil {
		return 0, false
	}

	return time.Duration(delay) * time.Second, true
}
