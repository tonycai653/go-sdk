// Package defaults is a collection of helpers to retrieve the SDK's default
// configuration and handlers.
//
// Generally this package shouldn't be used directly.
// This package is useful when you need to reset the defaults
// of configuration for service client to the SDK defaults before setting
// additional parameters.
package defaults

import (
	"net/http"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/corehandlers"
	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/defs"
	"github.com/qiniu/go-sdk/qiniu/request"
)

// A Defaults provides a collection of default values for SDK clients.
type Defaults struct {
	Config   *qiniu.Config
	Handlers request.Handlers
}

// Get returns the SDK's default values with Config and handlers pre-configured.
func Get() Defaults {
	cfg := Config()
	handlers := Handlers()

	return Defaults{
		Config:   cfg,
		Handlers: handlers,
	}
}

// Config returns the default configuration without credentials.
// To retrieve a config with credentials also included use
// `defaults.Get().Config` instead.
//
// Generally you shouldn't need to use this method directly, but
// is available if you need to reset the configuration of an
// existing service client or session.
func Config() *qiniu.Config {
	return qiniu.NewConfig().
		WithHTTPClient(http.DefaultClient).
		WithMaxRetries(qiniu.UseServiceDefaultRetries).
		WithLogger(qiniu.NewDefaultLogger()).
		WithLogLevel(qiniu.LogOff).
		WithRsHost(defs.DefaultRsHost).
		WithRsfHost(defs.DefaultRsfHost).
		WithAPIHost(defs.DefaultAPIHost).
		WithUCHost(defs.DefaultUcHost)
}

// Handlers returns the default request handlers.
//
// Generally you shouldn't need to use this method directly, but
// is available if you need to reset the request handlers of an
// existing service client or session.
func Handlers() request.Handlers {
	var handlers request.Handlers

	handlers.Build.PushBackNamed(corehandlers.SDKVersionUserAgentHandler)
	handlers.Build.PushBackNamed(corehandlers.AddHostExecEnvUserAgentHander)
	handlers.Build.AfterEachFn = request.HandlerListStopOnError
	handlers.Sign.PushBackNamed(corehandlers.BuildContentLengthHandler)
	handlers.Sign.AfterEachFn = request.HandlerListStopOnError
	handlers.Send.PushBackNamed(corehandlers.SendHandler)
	handlers.AfterRetry.PushBackNamed(corehandlers.AfterRetryHandler)
	handlers.ValidateResponse.PushBackNamed(corehandlers.ValidateResponseHandler)
	handlers.Complete.PushBackNamed(corehandlers.CompleteHandler)

	return handlers
}

// CredChain returns the default credential chain.
//
// Generally you shouldn't need to use this method directly, but
// is available if you need to reset the credentials of an
// existing service client or session's Config.
func CredChain(cfg *qiniu.Config, handlers request.Handlers) *credentials.Credentials {
	return credentials.NewCredentials(&credentials.ChainProvider{
		VerboseErrors: qiniu.BoolValue(cfg.CredentialsChainVerboseErrors),
		Providers:     CredProviders(cfg, handlers),
	})
}

// CredProviders returns the slice of providers used in
// the default credential chain.
//
// For applications that need to use some other provider (for example use
// different  environment variables for legacy reasons) but still fall back
// on the default chain of providers. This allows that default chaint to be
// automatically updated
func CredProviders(cfg *qiniu.Config, handlers request.Handlers) []credentials.Provider {
	return []credentials.Provider{
		&credentials.EnvProvider{},
	}
}
