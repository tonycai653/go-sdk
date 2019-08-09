package session

import (
	"os"
	"strings"
	"testing"
)

func TestEnvHosts(t *testing.T) {
	checkFatal(t, os.Setenv("QINIU_ACCESS_KEY", "ak"))
	checkFatal(t, os.Setenv("QINIU_SECRET_KEY", "skid"))
	checkFatal(t, os.Setenv("QINIU_RS_HOST", "rs"))
	checkFatal(t, os.Setenv("QINIU_RSF_HOST", "rsf"))
	checkFatal(t, os.Setenv("QINIU_API_HOST", "api"))
	checkFatal(t, os.Setenv("QINIU_UC_HOST", "uc"))
	checkFatal(t, os.Setenv("QINIU_Z1_UP_HOSTS", "z1up1,z1up2"))
	checkFatal(t, os.Setenv("QINIU_Z1_RS_HOST", "z1rs"))
	checkFatal(t, os.Setenv("QINIU_Z1_RSF_HOST", "z1rsf"))
	checkFatal(t, os.Setenv("QINIU_Z1_API_HOST", "z1api"))
	checkFatal(t, os.Setenv("QINIU_Z1_IO_HOST", "z1io"))
	checkFatal(t, os.Setenv("QINIU_Z1_ACC_UP_HOSTS", "z1accup1,z1accup2"))

	envConfig := loadSharedEnvConfig()

	t.Run("env access secret key", func(t *testing.T) {
		if envConfig.Creds.AccessKey != "ak" {
			t.Errorf("Expected AccessKey = `%s`, Got = `%s`\n", "ak", envConfig.Creds.AccessKey)
		}
		if string(envConfig.Creds.SecretKey) != "skid" {
			t.Errorf("Expected AccessKey = `%s`, Got = `%s`\n", "skid", string(envConfig.Creds.SecretKey))
		}
	})

	t.Run("env global hosts", func(t *testing.T) {
		if envConfig.ApiHost != "api" {
			t.Errorf("Expected API HOST = `%s`, but Got = `%s`\n", "api", envConfig.ApiHost)
		}
		if envConfig.RsHost != "rs" {
			t.Errorf("Expected RS HOST = `%s`, but Got = `%s`\n", "api", envConfig.RsHost)
		}
		if envConfig.RsfHost != "rsf" {
			t.Errorf("Expected RSF HOST = `%s`, but Got = `%s`\n", "api", envConfig.RsfHost)
		}
		if envConfig.UcHost != "uc" {
			t.Errorf("Expected UC HOST = `%s`, but Got = `%s`\n", "api", envConfig.UcHost)
		}
	})

	t.Run("env zone hosts", func(t *testing.T) {
		if envConfig.Z1.RsHost != "z1rs" {
			t.Errorf("Expected Z1 RS HOST = `%s`, but Got = `%s`\n", "z1rs", envConfig.Z1.RsHost)
		}
		if envConfig.Z1.RsfHost != "z1rsf" {
			t.Errorf("Expected Z1 RSF HOST = `%s`, but Got = `%s`\n", "z1rsf", envConfig.Z1.RsfHost)
		}
		if envConfig.Z1.ApiHost != "z1api" {
			t.Errorf("Expected Z1 API HOST = `%s`, but Got = `%s`\n", "z1api", envConfig.Z1.ApiHost)
		}
		if envConfig.Z1.IoHost != "z1io" {
			t.Errorf("Expected Z1 IO HOST = `%s`, but Got = `%s`\n", "z1io", envConfig.Z1.IoHost)
		}
		if strings.Join(envConfig.Z1.UpHosts, ",") != "z1up1,z1up2" {
			t.Errorf("Expected Z1 UP HOSTS = `%s`, but Got = `%s`\n", "z1up1,z1up2", strings.Join(envConfig.Z1.UpHosts, ","))
		}
		if strings.Join(envConfig.Z1.AccUpHosts, ",") != "z1accup1,z1accup2" {
			t.Errorf("Expected Z1 ACC UP HOSTS = `%s`, but Got = `%s`\n", "z1accup1,z1accup2", strings.Join(envConfig.Z1.AccUpHosts, ","))
		}
	})

}

func checkFatal(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Expected nil error, but Got: %#v\n", err)
	}
}
