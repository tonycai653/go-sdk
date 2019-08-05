package session

import (
	"fmt"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/qerr"
)

// valid credential source values
const (
	credSourceEnvironment = "Environment"
)

func resolveCredentials(cfg *qiniu.Config,
	envCfg envConfig, sharedCfg sharedConfig,
	sessOpts Options,
) (*credentials.Credentials, error) {

	// Credentials from environment variables
	if len(envCfg.Creds.AccessKey) > 0 {
		return credentials.NewStaticCredentialsFromCreds(envCfg.Creds), nil
	}

	// Fallback to the "default" credential resolution chain.
	return resolveCredsFromProfile(cfg, envCfg, sharedCfg, sessOpts)
}

func resolveCredsFromProfile(cfg *qiniu.Config,
	envCfg envConfig, sharedCfg sharedConfig,
	sessOpts Options) (*credentials.Credentials, error) {

	if len(sharedCfg.Creds.AccessKey) > 0 {
		// Static Credentials from Shared Config/Credentials file.
		return credentials.NewStaticCredentialsFromCreds(
			sharedCfg.Creds,
		), nil
	}
	// Fallback to default credentials provider, include mock errors
	// for the credential chain so user can identify why credentials
	// failed to be retrieved.
	return credentials.NewCredentials(&credentials.ChainProvider{
		VerboseErrors: qiniu.BoolValue(cfg.CredentialsChainVerboseErrors),
		Providers: []credentials.Provider{
			&credProviderError{
				Err: qerr.New("EnvAccessKeyNotFound",
					"failed to find credentials in the environment.", nil),
			},
			&credProviderError{
				Err: qerr.New("SharedCredsLoad",
					fmt.Sprintf("failed to load profile"), nil),
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
func (c credProviderError) IsExpired() bool {
	return true
}
