package credentials

import (
	"os"

	"github.com/qiniu/go-sdk/qiniu/qerr"
)

// EnvProviderName provides a name of Env provider
const EnvProviderName = "EnvProvider"

var (
	// ErrAccessKeyIDNotFound is returned when the QINIU Access Key ID can't be
	// found in the process's environment.
	ErrAccessKeyIDNotFound = qerr.New("EnvAccessKeyNotFound", "QINIU_ACCESS_KEY_ID or QINIU_ACCESS_KEY not found in environment", nil)

	// ErrSecretAccessKeyNotFound is returned when the QINIU Secret Access Key
	// can't be found in the process's environment.
	ErrSecretAccessKeyNotFound = qerr.New("EnvSecretNotFound", "QINIU_SECRET_ACCESS_KEY or QINIU_SECRET_KEY not found in environment", nil)
)

// A EnvProvider retrieves credentials from the environment variables of the
// running process. Environment credentials never expire.
//
// Environment variables used:
//
// * Access Key ID:     QINIU_ACCESS_KEY_ID or QINIU_ACCESS_KEY
//
// * Secret Access Key: QINIU_SECRET_ACCESS_KEY or QINIU_SECRET_KEY
type EnvProvider struct {
	retrieved bool
}

// NewEnvCredentials returns a pointer to a new Credentials object
// wrapping the environment variable provider.
func NewEnvCredentials() *Credentials {
	return NewCredentials(&EnvProvider{})
}

// Retrieve retrieves the keys from the environment.
func (e *EnvProvider) Retrieve() (Value, error) {
	e.retrieved = false

	id := os.Getenv("QINIU_ACCESS_KEY_ID")
	if id == "" {
		id = os.Getenv("QINIU_ACCESS_KEY")
	}

	secret := os.Getenv("QINIU_SECRET_ACCESS_KEY")
	if secret == "" {
		secret = os.Getenv("QINIU_SECRET_KEY")
	}

	if id == "" {
		return Value{ProviderName: EnvProviderName}, ErrAccessKeyIDNotFound
	}

	if secret == "" {
		return Value{ProviderName: EnvProviderName}, ErrSecretAccessKeyNotFound
	}

	e.retrieved = true
	return Value{
		AccessKey:    id,
		SecretKey:    []byte(secret),
		ProviderName: EnvProviderName,
	}, nil
}

// IsExpired returns if the credentials have been retrieved.
func (e *EnvProvider) IsExpired() bool {
	return !e.retrieved
}
