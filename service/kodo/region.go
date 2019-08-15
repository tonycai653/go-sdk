package kodo

var regions = map[string]Region{
	"z0": Region{
		Name: "z0",
		RegionDomains: RegionDomains{
			Hosts: []RegionDomain{
				RegionDomain{
					Up: UpDomainGroup{
						Src: DomainGroup{
							Main: []string{
								"up.qiniup.com",
								"up-nb.qiniup.com",
								"up-xs.qiniup.com",
							},
						},
						Acc: DomainGroup{
							Main: []string{
								"upload.qiniup.com",
								"upload-nb.qiniup.com",
								"upload-xs.qiniup.com",
							},
						},
					},
					Io: IoDomainGroup{
						Src: DomainGroup{
							Main: []string{"iovip.qbox.me"},
						},
					},
					Rs: RsDomainGroup{
						Src: DomainGroup{
							Main: []string{"rs.qbox.me"},
						},
					},
					Rsf: RsfDomainGroup{
						Src: DomainGroup{
							Main: []string{"rsf.qbox.me"},
						},
					},
					API: APIDomainGroup{
						Src: DomainGroup{
							Main: []string{"api.qiniu.com"},
						},
					},
				},
			},
		},
	},
	"z1": Region{
		Name: "z1",
		RegionDomains: RegionDomains{
			Hosts: []RegionDomain{
				RegionDomain{
					Up: UpDomainGroup{
						Src: DomainGroup{
							Main: []string{
								"up-z1.qiniup.com",
							},
						},
						Acc: DomainGroup{
							Main: []string{
								"upload-z1.qiniup.com",
							},
						},
					},
					Io: IoDomainGroup{
						Src: DomainGroup{
							Main: []string{"iovip-z1.qbox.me"},
						},
					},
					Rs: RsDomainGroup{
						Src: DomainGroup{
							Main: []string{"rs-z1.qbox.me"},
						},
					},
					Rsf: RsfDomainGroup{
						Src: DomainGroup{
							Main: []string{"rsf-z1.qbox.me"},
						},
					},
					API: APIDomainGroup{
						Src: DomainGroup{
							Main: []string{"api-z1.qiniu.com"},
						},
					},
				},
			},
		},
	},
	"z2": Region{
		Name: "z2",
		RegionDomains: RegionDomains{
			Hosts: []RegionDomain{
				RegionDomain{
					Up: UpDomainGroup{
						Src: DomainGroup{
							Main: []string{
								"up-z2.qiniup.com",
								"up-gz.qiniup.com",
								"up-fs.qiniup.com",
							},
						},
						Acc: DomainGroup{
							Main: []string{
								"upload-z2.qiniup.com",
								"upload-gz.qiniup.com",
								"upload-fs.qiniup.com",
							},
						},
					},
					Io: IoDomainGroup{
						Src: DomainGroup{
							Main: []string{"iovip-z2.qbox.me"},
						},
					},
					Rs: RsDomainGroup{
						Src: DomainGroup{
							Main: []string{"rs-z2.qbox.me"},
						},
					},
					Rsf: RsfDomainGroup{
						Src: DomainGroup{
							Main: []string{"rsf-z2.qbox.me"},
						},
					},
					API: APIDomainGroup{
						Src: DomainGroup{
							Main: []string{"api-z2.qiniu.com"},
						},
					},
				},
			},
		},
	},
	"as0": Region{
		Name: "as0",
		RegionDomains: RegionDomains{
			Hosts: []RegionDomain{
				RegionDomain{
					Up: UpDomainGroup{
						Src: DomainGroup{
							Main: []string{
								"up-as0.qiniup.com",
							},
						},
						Acc: DomainGroup{
							Main: []string{
								"upload-as0.qiniup.com",
							},
						},
					},
					Io: IoDomainGroup{
						Src: DomainGroup{
							Main: []string{"iovip-as0.qbox.me"},
						},
					},
					Rs: RsDomainGroup{
						Src: DomainGroup{
							Main: []string{"rs-as0.qbox.me"},
						},
					},
					Rsf: RsfDomainGroup{
						Src: DomainGroup{
							Main: []string{"rsf-as0.qbox.me"},
						},
					},
					API: APIDomainGroup{
						Src: DomainGroup{
							Main: []string{"api-as0.qiniu.com"},
						},
					},
				},
			},
		},
	},
	"na0": Region{
		Name: "na0",
		RegionDomains: RegionDomains{
			Hosts: []RegionDomain{
				RegionDomain{
					Up: UpDomainGroup{
						Src: DomainGroup{
							Main: []string{
								"up-na0.qiniup.com",
							},
						},
						Acc: DomainGroup{
							Main: []string{
								"upload-na0.qiniup.com",
							},
						},
					},
					Io: IoDomainGroup{
						Src: DomainGroup{
							Main: []string{"iovip-na0.qbox.me"},
						},
					},
					Rs: RsDomainGroup{
						Src: DomainGroup{
							Main: []string{"rs-na0.qbox.me"},
						},
					},
					Rsf: RsfDomainGroup{
						Src: DomainGroup{
							Main: []string{"rsf-na0.qbox.me"},
						},
					},
					API: APIDomainGroup{
						Src: DomainGroup{
							Main: []string{"api-na0.qiniu.com"},
						},
					},
				},
			},
		},
	},
}

// Region 是七牛存储空间所在的区域Host信息
// 包括上传域名么，加速上传域名，下载的存储入口等等
type Region struct {
	// 存储区域的名字
	// 合法的名字有z0, z1, z2, as0, na0
	// 分别表示华东， 华北， 华南， 东南亚， 北美
	Name string

	RegionDomains
}

// IsEmpty 判断Region是否为空
// 如果Name字段是空，那么返回true， 否则返回false
func (r Region) IsEmpty() bool {
	return r.RegionDomains.IsEmpty()
}

// GetDefaultRegion 根据区域的名称返回Region信息
// 如果regionName合法，也就是在（`z0`, `z1`, `z2`, `as0`, `na0`)中，
// 那么就从本地记录的信息返回, 否则返回空的Region
func GetDefaultRegion(regionName string) Region {
	if r, ok := regions[regionName]; ok {
		return r
	}
	return Region{}
}
