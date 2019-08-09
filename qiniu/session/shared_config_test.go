package session

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/qiniu/go-sdk/internal/ini"
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/definitions"
)

var (
	testConfigFilename      = filepath.Join("testdata", "shared_config")
	testConfigOtherFilename = filepath.Join("testdata", "shared_config_other")
)

func TestLoadSharedConfig(t *testing.T) {
	cases := []struct {
		Filenames []string
		Profile   string
		Expected  sharedConfig
		Err       error
	}{
		{
			Filenames: []string{testConfigFilename},
			Profile:   "profile",
			Expected: sharedConfig{
				Creds: credentials.Value{
					AccessKey:    "qiniu_access_key_id",
					SecretKey:    []byte("qiniu_secret_access_key"),
					ProviderName: fmt.Sprintf("SharedConfigCredentials: %s", testConfigFilename),
				},
			},
		},
		{
			Filenames: []string{testConfigFilename},
			Profile:   "z0",
			Expected: sharedConfig{
				Z0: definitions.Host{
					RsHost:     "rs",
					RsfHost:    "rsf",
					IoHost:     "io",
					ApiHost:    "api",
					UpHosts:    []string{"domain1", "domain2"},
					AccUpHosts: []string{"domain1", "domain2"},
				},
			},
		},
		{
			Filenames: []string{testConfigFilename},
			Profile:   "host",
			Expected: sharedConfig{
				RsHost:  "qiniu_rs_host",
				RsfHost: "qiniu_rsf_host",
				UcHost:  "qiniu_uc_host",
				ApiHost: "qiniu_api_host",
			},
		},
		{
			Filenames: []string{"file_not_exists"},
			Profile:   "default",
		},
		{
			Filenames: []string{testConfigOtherFilename, testConfigFilename},
			Profile:   "assume_role_w_creds",
			Expected: sharedConfig{
				Creds: credentials.Value{
					AccessKey:    "assume_role_w_creds_akid",
					SecretKey:    []byte("assume_role_w_creds_secret"),
					ProviderName: fmt.Sprintf("SharedConfigCredentials: %s", testConfigFilename),
				},
			},
		},
		{
			Filenames: []string{filepath.Join("testdata", "shared_config_invalid_ini")},
			Profile:   "profile_name",
			Err:       SharedConfigLoadError{Filename: filepath.Join("testdata", "shared_config_invalid_ini")},
		},
	}

	for i, c := range cases {
		cfg, err := loadSharedConfig([]string{c.Profile}, c.Filenames)
		if c.Err != nil {
			if e, a := c.Err.Error(), err.Error(); !strings.Contains(a, e) {
				t.Errorf("%d, expect %v, to contain %v", i, e, a)
			}
			continue
		}

		if err != nil {
			t.Errorf("%d, expect nil, %v", i, err)
		}
		if e, a := c.Expected, cfg; !reflect.DeepEqual(e, a) {
			t.Errorf("%d, expect %v, got %v", i, e, a)
		}
	}
}

func TestLoadSharedConfigFromFile(t *testing.T) {
	filename := testConfigFilename
	f, err := ini.OpenFile(filename)
	if err != nil {
		t.Fatalf("failed to load test config file, %s, %v", filename, err)
	}
	iniFile := sharedConfigFile{IniData: f, Filename: filename}

	cases := []struct {
		Profile  string
		Expected sharedConfig
		Err      error
	}{
		{
			Profile:  "partial_creds",
			Expected: sharedConfig{},
		},
		{
			Profile: "complete_creds",
			Expected: sharedConfig{
				Creds: credentials.Value{
					AccessKey:    "complete_creds_akid",
					SecretKey:    []byte("complete_creds_secret"),
					ProviderName: fmt.Sprintf("SharedConfigCredentials: %s", testConfigFilename),
				},
			},
		},
		{
			Profile:  "partial_assume_role",
			Expected: sharedConfig{},
		},
		{
			Profile: "does_not_exists",
			Err:     SharedConfigProfileNotExistsError{Profile: "does_not_exists"},
		},
	}

	for i, c := range cases {
		cfg := sharedConfig{}

		err := cfg.setFromIniFile(c.Profile, iniFile)
		if c.Err != nil {
			if e, a := c.Err.Error(), err.Error(); !strings.Contains(a, e) {
				t.Errorf("%d, expect %v, to contain %v", i, e, a)
			}
			continue
		}

		if err != nil {
			t.Errorf("%d, expect nil, %v", i, err)
		}
		if e, a := c.Expected, cfg; !reflect.DeepEqual(e, a) {
			t.Errorf("%d, expect %v, got %v", i, e, a)
		}
	}
}

func TestLoadSharedConfigIniFiles(t *testing.T) {
	cases := []struct {
		Filenames []string
		Expected  []sharedConfigFile
	}{
		{
			Filenames: []string{"not_exists", testConfigFilename},
			Expected: []sharedConfigFile{
				{Filename: testConfigFilename},
			},
		},
		{
			Filenames: []string{testConfigFilename, testConfigOtherFilename},
			Expected: []sharedConfigFile{
				{Filename: testConfigFilename},
				{Filename: testConfigOtherFilename},
			},
		},
	}

	for i, c := range cases {
		files, err := loadSharedConfigIniFiles(c.Filenames)
		if err != nil {
			t.Errorf("%d, expect nil, %v", i, err)
		}
		if e, a := len(c.Expected), len(files); e != a {
			t.Errorf("expect %v, got %v", e, a)
		}

		for i, expectedFile := range c.Expected {
			if e, a := expectedFile.Filename, files[i].Filename; e != a {
				t.Errorf("expect %v, got %v", e, a)
			}
		}
	}
}
