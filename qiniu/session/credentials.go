package session

import (
	"fmt"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/qerr"
)

const (
	credSourceEnvironment = "Environment"
)

func resolveCredentials(cfg *qiniu.Config,
	envCfg envConfig, sharedCfg sharedConfig,
	sessOpts Options,
) (*credentials.Credentials, error) {

	if len(envCfg.Creds.AccessKey) > 0 {
		return credentials.NewStaticCredentialsFromCreds(envCfg.Creds), nil
	}

	return resolveCredsFromConfigFile(cfg, envCfg, sharedCfg, sessOpts)
}

func resolveCredsFromConfigFile(cfg *qiniu.Config,
	envCfg envConfig, sharedCfg sharedConfig,
	sessOpts Options) (*credentials.Credentials, error) {

	if len(sharedCfg.Creds.AccessKey) > 0 {
		return credentials.NewStaticCredentialsFromCreds(
			sharedCfg.Creds,
		), nil
	}
	return credentials.NewCredentials(&credentials.ChainProvider{
		VerboseErrors: qiniu.BoolValue(cfg.CredentialsChainVerboseErrors),
		Providers: []credentials.Provider{
			&credProviderError{
				Err: qerr.New("EnvAccessKeyNotFound",
					"failed to find credentials in the environment.", nil),
			},
			&credProviderError{
				Err: qerr.New("SharedCredsLoad",
					fmt.Sprintf("failed to load config file"), nil),
			},
		},
	}), nil
}

type credProviderError struct {
	Err error
}

var emptyCreds = credentials.Value{}

func (c credProviderError) Retrieve() (credentials.Value, error) {
	return credentials.Value{}, c.Err
}
