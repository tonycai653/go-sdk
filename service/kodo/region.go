package kodo

import (
	"github.com/qiniu/go-sdk/qiniu/definitions"
)

var regions = map[string]Region{
	"z0": Region{
		Name: "z0",
		SrcUpHosts: []string{
			"up.qiniup.com",
			"up-nb.qiniup.com",
			"up-xs.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload.qiniup.com",
			"upload-nb.qiniup.com",
			"upload-xs.qiniup.com",
		},
		RsHost:    "rs.qbox.me",
		RsfHost:   "rsf.qbox.me",
		ApiHost:   "api.qiniu.com",
		IovipHost: "iovip.qbox.me",
	},
	"z1": Region{
		Name: "z1",
		SrcUpHosts: []string{
			"up-z1.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload-z1.qiniup.com",
		},
		RsHost:    "rs-z1.qbox.me",
		RsfHost:   "rsf-z1.qbox.me",
		ApiHost:   "api-z1.qiniu.com",
		IovipHost: "iovip-z1.qbox.me",
	},
	"z2": Region{
		Name: "z2",
		SrcUpHosts: []string{
			"up-z2.qiniup.com",
			"up-gz.qiniup.com",
			"up-fs.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload-z2.qiniup.com",
			"upload-gz.qiniup.com",
			"upload-fs.qiniup.com",
		},
		RsHost:  "rs-z2.qbox.me",
		RsfHost: "rsf-z2.qbox.me",
		ApiHost: "api-z2.qiniu.com",
	},
	"as0": Region{
		Name: "as0",
		SrcUpHosts: []string{
			"up-as0.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload-as0.qiniup.com",
		},
		RsHost:    "rs-as0.qbox.me",
		RsfHost:   "rsf-as0.qbox.me",
		ApiHost:   "api-as0.qiniu.com",
		IovipHost: "iovip-as0.qbox.me",
	},
	"na0": Region{
		Name: "na0",
		SrcUpHosts: []string{
			"up-na0.qiniup.com",
		},
		CdnUpHosts: []string{
			"upload-na0.qiniup.com",
		},
		RsHost:    "rs-na0.qbox.me",
		RsfHost:   "rsf-na0.qbox.me",
		ApiHost:   "api-na0.qiniu.com",
		IovipHost: "iovip-na0.qbox.me",
	},
}

// Region 是七牛存储空间所在的区域Host信息
// 包括上传域名么，加速上传域名，下载的存储入口等等
type Region struct {
	// 存储区域的名字
	// 合法的名字有z0, z1, z2, as0, na0
	// 分别表示华东， 华北， 华南， 东南亚， 北美
	Name string

	definitions.Host
}

// GetRegion 根据区域的名称返回Region信息
// 如果regionName合法，也就是在（`z0`, `z1`, `z2`, `as0`, `na0`)中，
// 那么就从本地记录的信息返回
// 如果regionName不合法或者是空， 那么会忽略regionName的值， 直接根据bucket通过API请求获取该存储空间的信息
func GetRegion(accessKey, bucket, regionName string) Region {
	if r, ok := regions[regionName]; ok {
		return r
	}
	return getRegionFromRemote(accessKey, regionName)
}

func getRegionFromRemote(accessKey, regionName string) Region {
	return Region{}
}
