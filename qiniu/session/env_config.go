package session

import (
	"os"

	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/defaults"
)

// EnvProviderName provides a name of the provider when config is loaded from environment.
const EnvProviderName = "EnvConfigCredentials"

// envConfig is a collection of environment values the SDK will read
// setup config from. All environment values are optional. But some values
// such as credentials require multiple values to be complete or the values
// will be ignored.
type envConfig struct {
	// Environment configuration values. If set both Access Key ID and Secret Access
	// Key must be provided.
	//
	//	# Access Key ID
	//	QINIU_ACCESS_KEY_ID=AKID
	//	QINIU_ACCESS_KEY=AKID # only read if QINIU_ACCESS_KEY_ID is not set.
	//
	//	# Secret Access Key
	//	QINIU_SECRET_ACCESS_KEY=SECRET
	//	QINIU_SECRET_KEY=SECRET=SECRET # only read if QINIU_SECRET_ACCESS_KEY is not set.
	Creds credentials.Value

	// Shared credentials file path can be set to instruct the SDK to use an alternate
	// file for the shared credentials. If not set the file will be loaded from
	// $HOME/.qiniu/credentials on Linux/Unix based systems, and
	// %USERPROFILE%\.qiniu\credentials on Windows.
	//
	//	QINIU_SHARED_CREDENTIALS_FILE=$HOME/my_shared_credentials
	SharedCredentialsFile string

	// Shared config file path can be set to instruct the SDK to use an alternate
	// file for the shared config. If not set the file will be loaded from
	// $HOME/.qiniu/config on Linux/Unix based systems, and
	// %USERPROFILE%\.qiniu\config on Windows.
	//
	//	QINIU_CONFIG_FILE=$HOME/my_shared_config
	SharedConfigFile string

	// 环境变量对如下Host的配置
	// 环境变量： QINIU_RS_HOST
	RsHost string

	// 环境变量： QINIU_RSF_HOST
	RsfHost string

	// 环境变量: QINIU_API_HOST
	ApiHost string

	// 环境变量: QINIU_UC_HOST
	UcHost string
}

var (
	rsHostEnvKey = []string{
		"QINIU_RS_HOST",
	}
	rsfHostEnvKey = []string{
		"QINIU_RSF_HOST",
	}
	ucHostEnvKey = []string{
		"QINIU_UC_HOST",
	}
	apiHostEnvKey = []string{
		"QINIU_API_HOST",
	}
	credAccessEnvKey = []string{
		"QINIU_ACCESS_KEY_ID",
		"QINIU_ACCESS_KEY",
	}
	credSecretEnvKey = []string{
		"QINIU_SECRET_ACCESS_KEY",
		"QINIU_SECRET_KEY",
	}
	sharedCredsFileEnvKey = []string{
		"QINIU_SHARED_CREDENTIALS_FILE",
	}
	sharedConfigFileEnvKey = []string{
		"QINIU_CONFIG_FILE",
	}
)

// loadSharedEnvConfig retrieves the SDK's environment configuration, and the
// SDK shared config. See `envConfig` for the values that will be retrieved.
//
// Loads the shared configuration in addition to the SDK's specific configuration.
func loadSharedEnvConfig() envConfig {
	cfg := envConfig{}

	setFromEnvVal(&cfg.Creds.AccessKey, credAccessEnvKey)
	cfg.Creds.SecretKey = sliceFromEnvVal(credSecretEnvKey)
	setFromEnvVal(&cfg.RsHost, rsHostEnvKey)
	setFromEnvVal(&cfg.RsfHost, rsfHostEnvKey)
	setFromEnvVal(&cfg.ApiHost, apiHostEnvKey)
	setFromEnvVal(&cfg.UcHost, ucHostEnvKey)

	// Require logical grouping of credentials
	if len(cfg.Creds.AccessKey) == 0 || len(cfg.Creds.SecretKey) == 0 {
		cfg.Creds = credentials.Value{}
	} else {
		cfg.Creds.ProviderName = EnvProviderName
	}

	setFromEnvVal(&cfg.SharedCredentialsFile, sharedCredsFileEnvKey)
	setFromEnvVal(&cfg.SharedConfigFile, sharedConfigFileEnvKey)

	if len(cfg.SharedCredentialsFile) == 0 {
		cfg.SharedCredentialsFile = defaults.SharedCredentialsFilename()
	}
	if len(cfg.SharedConfigFile) == 0 {
		cfg.SharedConfigFile = defaults.SharedConfigFilename()
	}

	return cfg
}

func sliceFromEnvVal(keys []string) []byte {
	for _, k := range keys {
		if v := os.Getenv(k); len(v) > 0 {
			s := make([]byte, len(v))
			copy(s, v)
			return s
		}
	}
	return nil
}

func setFromEnvVal(dst *string, keys []string) {
	for _, k := range keys {
		if v := os.Getenv(k); len(v) > 0 {
			*dst = v
			break
		}
	}
}
