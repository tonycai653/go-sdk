package defs

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
	PB = 1024 * GB
)

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