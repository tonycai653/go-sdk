// Package definitions 定义了一些类型，会被多个包使用
// 该包不依赖于任何其他的包
package definitions

// Host定义了存储区域HOSTS信息
type Host struct {
	// 上传入口
	UpHosts []string

	// 加速上传入口
	AccUpHosts []string

	// 获取文件信息入口
	RsHost string

	// bucket列举入口
	RsfHost string

	ApiHost string

	// 存储io 入口
	IoHost string
}
