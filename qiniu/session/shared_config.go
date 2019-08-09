package session

import (
	"fmt"
	"strings"

	"github.com/qiniu/go-sdk/internal/ini"
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/definitions"
	"github.com/qiniu/go-sdk/qiniu/qerr"
)

const (
	// Static Credentials group
	accessKeyIDKey  = `qiniu_access_key_id`     // group required
	secretAccessKey = `qiniu_secret_access_key` // group required

	rsHostKey  = "qiniu_rs_host"
	rsfHostKey = "qiniu_rsf_host"
	apiHostKey = "qiniu_api_host"
	ucHostKey  = "qiniu_uc_host"

	// DefaultSharedConfigProfile is the default profile to be used when
	// loading configuration from the config files if another profile name
	// is not provided.
	DefaultSharedConfigProfile = `default`
)

var zoneKeys = map[string]string{
	"rs":  "qiniu_rs_host",
	"rsf": "qiniu_rsf_host",
	"api": "qiniu_api_host",
	"io":  "qiniu_io_host",
	"up":  "qiniu_up_hosts",
	"acc": "qiniu_acc_up_hosts",
}

// sharedConfig represents the configuration fields of the SDK config files.
type sharedConfig struct {
	// Credentials values from the config file. Both qiniu_access_key_id
	// and qiniu_secret_access_key must be provided together in the same file
	// to be considered valid. The values will be ignored if not a complete group.
	//
	//	qiniu_access_key_id
	//	qiniu_secret_access_key
	Creds credentials.Value

	// Hosts配置
	RsHost  string
	RsfHost string
	ApiHost string
	UcHost  string

	Z0  definitions.Host
	Z1  definitions.Host
	Z2  definitions.Host
	Na0 definitions.Host
	As0 definitions.Host
}

var defaultSections = []string{"profile", "host"}

type sharedConfigFile struct {
	Filename string
	IniData  ini.Sections
}

// loadSharedConfig retrieves the configuration from the list of files
// using the profile provided. The order the files are listed will determine
// precedence. Values in subsequent files will overwrite values defined in
// earlier files.
//
// For example, given two files A and B. Both define credentials. If the order
// of the files are A then B, B's credential values will be used instead of A's.
//
// See sharedConfig.setFromFile for information how the config files
// will be loaded.
func loadSharedConfig(profiles []string, filenames []string) (sharedConfig, error) {

	files, err := loadSharedConfigIniFiles(filenames)
	if err != nil {
		return sharedConfig{}, err
	}

	cfg := sharedConfig{}
	for _, profile := range profiles {
		if len(profile) == 0 {
			continue
		}
		_ = cfg.setFromIniFiles(profile, files)
	}
	return cfg, nil
}

// loadSharedConfigDefaultSections retrieves the configuration from the list of files
// using the default sections. The order the files are listed will determine
// precedence. Values in subsequent files will overwrite values defined in
// earlier files.
//
// For example, given two files A and B. Both define credentials. If the order
// of the files are A then B, B's credential values will be used instead of A's.
//
// See sharedConfig.setFromFile for information how the config files
// will be loaded.
func loadSharedConfigDefaultSections(filenames []string) (sharedConfig, error) {
	return loadSharedConfig(defaultSections, filenames)
}

func loadSharedConfigIniFiles(filenames []string) ([]sharedConfigFile, error) {
	files := make([]sharedConfigFile, 0, len(filenames))

	for _, filename := range filenames {
		sections, err := ini.OpenFile(filename)
		if aerr, ok := err.(qerr.Error); ok && aerr.Code() == ini.ErrCodeUnableToReadFile {
			// Skip files which can't be opened and read for whatever reason
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

func (cfg *sharedConfig) setFromIniFiles(profile string, files []sharedConfigFile) error {
	// Trim files from the list that don't exist.
	for _, f := range files {
		if err := cfg.setFromIniFile(profile, f); err != nil {
			if _, ok := err.(SharedConfigProfileNotExistsError); ok {
				// Ignore proviles missings
				continue
			}
			return err
		}
	}

	return nil
}

// setFromFile loads the configuration from the file.
// A sharedConfig pointer type value is used so that
// multiple config file loadings can be chained.
//
// Only loads complete logically grouped values, and will not set fields in cfg
// for incomplete grouped values in the config. Such as credentials. For example
// if a config file only includes qiniu_access_key_id but no qiniu_secret_access_key
// the qiniu_access_key_id will be ignored.
func (cfg *sharedConfig) setFromIniFile(profile string, file sharedConfigFile) error {
	section, ok := file.IniData.GetSection(profile)
	if !ok {
		return SharedConfigProfileNotExistsError{Profile: profile, Err: nil}
	}
	switch profile {
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
	case "host":
		cfg.hostsFromSection(section)
	default:
		cfg.credsFromSection(section, file.Filename)
	}
	return nil
}

func zoneHostFromSection(section ini.Section) *definitions.Host {
	h := definitions.Host{}
	h.RsHost = section.String(zoneKeys["rs"])
	h.RsfHost = section.String(zoneKeys["rsf"])
	h.IoHost = section.String(zoneKeys["io"])
	h.ApiHost = section.String(zoneKeys["api"])
	h.UpHosts = strings.Split(section.String(zoneKeys["up"]), ",")
	h.AccUpHosts = strings.Split(section.String(zoneKeys["acc"]), ",")

	return &h
}

// hostsFromSection 从ini.Section中获取hosts信息
func (cfg *sharedConfig) hostsFromSection(section ini.Section) {
	cfg.RsHost = section.String(rsHostKey)
	cfg.RsfHost = section.String(rsfHostKey)
	cfg.UcHost = section.String(ucHostKey)
	cfg.ApiHost = section.String(apiHostKey)
}

// credsFromSection 从section中获取密钥信息， 设置cfg.Creds字段
func (cfg *sharedConfig) credsFromSection(section ini.Section, filename string) {

	// Shared Credentials
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

// SharedConfigLoadError is an error for the shared config file failed to load.
type SharedConfigLoadError struct {
	Filename string
	Err      error
}

// Code is the short id of the error.
func (e SharedConfigLoadError) Code() string {
	return "SharedConfigLoadError"
}

// Message is the description of the error
func (e SharedConfigLoadError) Message() string {
	return fmt.Sprintf("failed to load config file, %s", e.Filename)
}

// OrigErr is the underlying error that caused the failure.
func (e SharedConfigLoadError) OrigErr() error {
	return e.Err
}

// Error satisfies the error interface.
func (e SharedConfigLoadError) Error() string {
	return qerr.SprintError(e.Code(), e.Message(), "", e.Err)
}

// SharedConfigProfileNotExistsError is an error for the shared config when
// the profile was not find in the config file.
type SharedConfigProfileNotExistsError struct {
	Profile string
	Err     error
}

// Code is the short id of the error.
func (e SharedConfigProfileNotExistsError) Code() string {
	return "SharedConfigProfileNotExistsError"
}

// Message is the description of the error
func (e SharedConfigProfileNotExistsError) Message() string {
	return fmt.Sprintf("failed to get profile, %s", e.Profile)
}

// OrigErr is the underlying error that caused the failure.
func (e SharedConfigProfileNotExistsError) OrigErr() error {
	return e.Err
}

// Error satisfies the error interface.
func (e SharedConfigProfileNotExistsError) Error() string {
	return qerr.SprintError(e.Code(), e.Message(), "", e.Err)
}
