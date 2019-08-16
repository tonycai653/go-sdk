package kodo

import (
	"github.com/qiniu/go-sdk/qiniu/defs"
)

const (
	// DefaultUploadPartSize 定义了分片上传每一个片的默认大小
	DefaultUploadPartSize = 4 * defs.MB

	// DefaultUploadConcurrency 定义了分片上传默认的goroutines数目
	DefaultUploadConcurrency = 5
)

// PartUploadInput 定义了分片上传的输入结构体。
type PartUploadInput struct {
	// Bucket 是要上传的存储空间名字
	Bucket string

	// Filename 是要上传保存在存储中的文件名
	Filename string

	// UploadID 唯一地标识一个文件分片上传的过程
	UploadID string

	// PartIndex是要上传的块的索引值
	PartIndex int
}

// PartUploadInit 实现七牛v2版本分片上传的初始化部分
func (u *Kodo) PartUploadInit(bucketName, filename string) error {
	return nil
}

// UploadPart 上传文件的块到存储空间
func (u *Kodo) UploadPart(input *PartUploadInput, out interface{}) error {
	return nil
}
