package credentials

import (
	"github.com/qiniu/go-sdk/qiniu/qerr"
)

var (
	// ErrNoValidProvidersFoundInChain Is returned when there are no valid
	// providers in the ChainProvider.
	//
	// This has been deprecated. For verbose error messaging set
	// api.Config.CredentialsChainVerboseErrors to true.
	ErrNoValidProvidersFoundInChain = qerr.New("NoCredentialProviders",
		`no valid providers in chain. Deprecated.
	For verbose messaging see api.Config.CredentialsChainVerboseErrors`,
		nil)
)

// A ChainProvider will search for a provider which returns credentials
// and cache that provider until Retrieve is called again.
//
// The ChainProvider provides a way of chaining multiple providers together
// which will pick the first available using priority order of the Providers
// in the list.
//
// If none of the Providers retrieve valid credentials Value, ChainProvider's
// Retrieve() will return the error ErrNoValidProvidersFoundInChain.
//
// If a Provider is found which returns valid credentials Value ChainProvider
// will cache that Provider.
type ChainProvider struct {
	Providers     []Provider
	curr          Provider
	VerboseErrors bool
}

// NewChainCredentials returns a pointer to a new Credentials object
// wrapping a chain of providers.
func NewChainCredentials(providers []Provider) *Credentials {
	return NewCredentials(&ChainProvider{
		Providers: append([]Provider{}, providers...),
	})
}

// Retrieve returns the credentials value or error if no provider returned
// without error.
//
// If a provider is found it will be cached
func (c *ChainProvider) Retrieve() (Value, error) {
	var errs []error
	for _, p := range c.Providers {
		creds, err := p.Retrieve()
		if err == nil {
			c.curr = p
			return creds, nil
		}
		errs = append(errs, err)
	}
	c.curr = nil

	var err error
	err = ErrNoValidProvidersFoundInChain
	if c.VerboseErrors {
		err = qerr.NewBatchError("NoCredentialProviders", "no valid providers in chain", errs)
	}
	return Value{}, err
}
