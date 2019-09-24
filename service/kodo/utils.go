package kodo

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/qiniu/go-sdk/qiniu/qerr"
)

// DecodeUpToken 解析上传token, 返回accessKey和上传策略指针
// 如果上传token不合法， 返回相应的错误
func DecodeUpToken(upToken string) (accessKey string, putPolicy *PutPolicy, err error) {
	splits := strings.SplitN(upToken, ":", 3)
	if len(splits) != 3 {
		err = qerr.New(ErrInvalidUptoken, "invalid upload token: "+upToken, nil)
		return
	}
	accessKey = splits[0]
	bs, bderr := base64.URLEncoding.DecodeString(splits[2])
	if bderr != nil {
		err = bderr
		return
	}
	p := PutPolicy{}
	derr := json.NewDecoder(strings.NewReader(string(bs))).Decode(&p)
	if derr != nil {
		err = derr
	}
	putPolicy = &p
	return
}
