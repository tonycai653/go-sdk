package qiniu

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/qiniu/go-sdk/qiniu/qerr"
)

const (
	// ErrInvalidHostFormat 错误的域名格式
	ErrInvalidHostFormat = "InvalidHostFormatError"

	// ErrInvalidUptoken 无效的上传token
	ErrInvalidUptoken = "InvalidUptokenError"
)

// EncodedEntry 生成URL Safe Base64编码的 Entry
func EncodedEntry(bucket, key string) string {
	entry := fmt.Sprintf("%s:%s", bucket, key)
	return base64.URLEncoding.EncodeToString([]byte(entry))
}

// NormalizeHost 规范化host， 去掉host中可能带的scheme信息，返回纯host, 和scheme信息
// 返回的信息分别是host, scheme, error
func NormalizeHost(host string) (string, string, error) {
	splits := strings.SplitN(host, "://", 2)
	if len(splits) > 2 {
		return "", "", qerr.New(ErrInvalidHostFormat, "invalid host format: "+host, nil)
	}
	if len(splits) == 1 {
		return splits[0], "", nil
	}
	return splits[1], splits[0], nil
}
