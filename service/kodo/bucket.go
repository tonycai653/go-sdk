package kodo

import (
	"fmt"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/defs"
	"github.com/qiniu/go-sdk/qiniu/request"
)

// Stat 发起stat请求， 返回存储在七牛空间的文件信息
func (c *Kodo) Stat(bucket, key string) (*FileInfo, error) {
	req, fileInfo := c.StatRequest(bucket, key)
	return fileInfo, req.Send()
}

// StatRequest 返回request.Request指针， 用于发起stat接口请求
func (c *Kodo) StatRequest(bucket, key string) (req *request.Request, info *FileInfo) {
	op := &request.API{
		Scheme:      "http",
		Path:        fmt.Sprintf("/stat/%s", qiniu.EncodedEntry(bucket, key)),
		Method:      "POST",
		Host:        c.Config.RsHost,
		ContentType: defs.CONTENT_TYPE_JSON,
		TokenType:   credentials.TokenQBox,
		APIName:     "stat",
		ServiceName: ServiceName,
	}
	info = &FileInfo{}
	req = c.newRequest(op, nil, info)
	return
}
