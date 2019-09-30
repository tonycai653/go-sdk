package session

import (
	"fmt"

	"github.com/qiniu/go-sdk/internal/ini"
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/qerr"
)

const (
	accessKeyIDKey  = `qiniu_access_key_id`
	secretAccessKey = `qiniu_secret_access_key`

	rsHostKey  = "qiniu_rs_host"
	rsfHostKey = "qiniu_rsf_host"
	apiHostKey = "qiniu_api_host"
	ucHostKey  = "qiniu_uc_host"
)

/*
var zoneKeys = map[string]string{
	"rs":  "qiniu_rs_host",
	"rsf": "qiniu_rsf_host",
	"api": "qiniu_api_host",
	"io":  "qiniu_io_host",
	"up":  "qiniu_up_hosts",
	"acc": "qiniu_acc_up_hosts",
}
*/

// sharedConfig 代表SDK配置文件的配置项
type sharedConfig struct {
	// 从配置文件获取密钥信息， qiniu_access_key_id, qiniu_secret_access_key都要设置才行
	// 否则视为无效的字段，相当于没有设置密钥信息
	//
	//	qiniu_access_key_id
	//	qiniu_secret_access_key
	Creds credentials.Value

	// Hosts配置
	RsHost  string
	RsfHost string
	APIHost string
	UcHost  string

	/*
		Z0  defs.Host
		Z1  defs.Host
		Z2  defs.Host
		Na0 defs.Host
		As0 defs.Host
	*/
}

var defaultSections = []string{"credentials", "host"}

type sharedConfigFile struct {
	Filename string
	IniData  ini.Sections
}

func loadSharedConfig(sections []string, filenames []string) (sharedConfig, error) {

	files, err := loadSharedConfigIniFiles(filenames)
	if err != nil {
		return sharedConfig{}, err
	}

	cfg := sharedConfig{}
	for _, section := range sections {
		if len(section) == 0 {
			continue
		}
		_ = cfg.setFromIniFiles(section, files)
	}
	return cfg, nil
}

// loadSharedConfigDefaultSections 从配置文件列表中获取配置信息， 配置文件在列表中的顺序
// 决定了配置项的优先级， 后面的配置文件会覆盖前面配置文件的值
//
// 比如A，和B文件都定义了密钥信息， 如果A文件在B文件之前， 那么会使用B文件的信息
func loadSharedConfigDefaultSections(filenames []string) (sharedConfig, error) {
	return loadSharedConfig(defaultSections, filenames)
}

func loadSharedConfigIniFiles(filenames []string) ([]sharedConfigFile, error) {
	files := make([]sharedConfigFile, 0, len(filenames))

	for _, filename := range filenames {
		sections, err := ini.OpenFile(filename)
		if aerr, ok := err.(qerr.Error); ok && aerr.Code() == ini.ErrCodeUnableToReadFile {
			continue
		} else if err != nil {
			return nil, SharedConfigLoadError{Filename: filename, Err: err}
		}

		files = append(files, sharedConfigFile{
			Filename: filename, IniData: sections,
		})
	}

	return files, nil
}

func (cfg *sharedConfig) setFromIniFiles(section string, files []sharedConfigFile) error {
	for _, f := range files {
		if err := cfg.setFromIniFile(section, f); err != nil {
			if _, ok := err.(SharedConfigSectionNotExistsError); ok {
				continue
			}
			return err
		}
	}
	return nil
}

// setFromIniFile 从文件中加载配置信息
// 对于逻辑上是一组信息必须完备的字段， 如果一组中的某个字段没有设置，那么这一组字段都不会设置
// 比如密钥信息，如果只配置了qiniu_access_key_id, 但是没有配置qiniu_secret_access_key， 或者反之，
// 那么这两个字段都被忽略，密钥信息相当于没有配置
func (cfg *sharedConfig) setFromIniFile(section string, file sharedConfigFile) error {
	sectionStruct, ok := file.IniData.GetSection(section)
	if !ok {
		return SharedConfigSectionNotExistsError{Section: section, Err: nil}
	}
	switch section {
	/*
		case "z0":
			h := zoneHostFromSection(section)
			cfg.Z0 = *h
		case "z1":
			h := zoneHostFromSection(section)
			cfg.Z1 = *h
		case "z2":
			h := zoneHostFromSection(section)
			cfg.Z2 = *h
		case "na0":
			h := zoneHostFromSection(section)
			cfg.Na0 = *h
		case "as0":
			h := zoneHostFromSection(section)
			cfg.As0 = *h
	*/
	case "host":
		cfg.hostsFromSection(sectionStruct)
	default:
		cfg.credsFromSection(sectionStruct, file.Filename)
	}
	return nil
}

/*
func zoneHostFromSection(section ini.Section) *defs.Host {
	h := defs.Host{}
	h.RsHost = section.String(zoneKeys["rs"])
	h.RsfHost = section.String(zoneKeys["rsf"])
	h.IoHost = section.String(zoneKeys["io"])
	h.APIHost = section.String(zoneKeys["api"])
	h.UpHosts = strings.Split(section.String(zoneKeys["up"]), ",")
	h.AccUpHosts = strings.Split(section.String(zoneKeys["acc"]), ",")

	return &h
}
*/

// hostsFromSection 从ini.Section中获取hosts信息
func (cfg *sharedConfig) hostsFromSection(section ini.Section) {
	cfg.RsHost = section.String(rsHostKey)
	cfg.RsfHost = section.String(rsfHostKey)
	cfg.UcHost = section.String(ucHostKey)
	cfg.APIHost = section.String(apiHostKey)
}

// credsFromSection 从section中获取密钥信息， 设置cfg.Creds字段
func (cfg *sharedConfig) credsFromSection(section ini.Section, filename string) {

	akid := section.String(accessKeyIDKey)
	secret := section.String(secretAccessKey)
	if len(akid) > 0 && len(secret) > 0 {
		cfg.Creds = credentials.Value{
			AccessKey:    akid,
			SecretKey:    []byte(secret),
			ProviderName: fmt.Sprintf("SharedConfigCredentials: %s", filename),
		}
	}
}

// SharedConfigLoadError 加载配置文件失败错误
type SharedConfigLoadError struct {
	Filename string
	Err      error
}

// Code 错误码
func (e SharedConfigLoadError) Code() string {
	return "SharedConfigLoadError"
}

// Message 是具体的错误描述信息
func (e SharedConfigLoadError) Message() string {
	return fmt.Sprintf("failed to load config file, %s", e.Filename)
}

// OrigErr 是导致这个错误的具体原因
func (e SharedConfigLoadError) OrigErr() error {
	return e.Err
}

// Error 满足error接口
func (e SharedConfigLoadError) Error() string {
	return qerr.SprintError(e.Code(), e.Message(), "", e.Err)
}

// SharedConfigSectionNotExistsError 代表配置文件的某个section不存在
type SharedConfigSectionNotExistsError struct {
	Section string
	Err     error
}

// Code 错误码
func (e SharedConfigSectionNotExistsError) Code() string {
	return "SharedConfigProfileNotExistsError"
}

// Message 错误的具体描述信息
func (e SharedConfigSectionNotExistsError) Message() string {
	return fmt.Sprintf("failed to get section, %s", e.Section)
}

// OrigErr 导致该错误的起因
func (e SharedConfigSectionNotExistsError) OrigErr() error {
	return e.Err
}

// Error 实现error接口
func (e SharedConfigSectionNotExistsError) Error() string {
	return qerr.SprintError(e.Code(), e.Message(), "", e.Err)
}
