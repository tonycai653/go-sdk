package defs

import (
	"fmt"
)

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
	PB = 1024 * GB
)

// Size 数据的大小
type Size int64

// String 返回数据大小的可读化显示
func (s Size) String() string {
	switch {
	case s < KB:
		return fmt.Sprintf("%dB", s)
	case s >= KB && s < MB:
		return fmt.Sprintf("%.2fKB", float64(s)/KB)
	case s >= MB && s < GB:
		return fmt.Sprintf("%.2fMB", float64(s)/MB)
	case s >= GB && s < PB:
		return fmt.Sprintf("%.2fGB", float64(s)/GB)
	default:
		return fmt.Sprintf("%.2fPB", float64(s)/PB)
	}
}

const (
	// DefaultRsHost 默认的RsHost
	DefaultRsHost = "rs.qiniu.com"

	// DefaultRsfHost 默认的RsfHost
	DefaultRsfHost = "rsf.qiniu.com"

	// DefaultAPIHost 默认的APIHost
	DefaultAPIHost = "api.qiniu.com"

	// DefaultUcHost 查询存储空间相关域名
	DefaultUcHost = "uc.qbox.me"

	// DefaultFormSize 默认的最大的可以使用表单方式上传的文件大小
	DefaultFormSize = 1 * MB
)
