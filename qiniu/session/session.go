package session

import (
	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/client"
	"github.com/qiniu/go-sdk/qiniu/corehandlers"
	"github.com/qiniu/go-sdk/qiniu/defaults"
	"github.com/qiniu/go-sdk/qiniu/request"
)

const (
	// ErrCodeSharedConfig represents an error that occurs in the shared
	// configuration logic
	ErrCodeSharedConfig = "SharedConfigErr"
)

// A Session provides a central location to create service clients from and
// store configurations and request handlers for those services.
//
// Sessions are safe to create service clients concurrently, but it is not safe
// to mutate the Session concurrently.
//
// The Session satisfies the service client's client.ConfigProvider.
type Session struct {
	Config   *qiniu.Config
	Handlers request.Handlers
}

// New returns a new Session created from SDK defaults, config files,
// environment, and user provided config files. Once the Session is created
// it can be mutated to modify the Config or Handlers. The Session is safe to
// be read concurrently, but it should not be written to concurrently.
//
// The shared config file (~/.qiniu/config) will be loaded in addition to
// the shared credentials file (~/.qiniu/credentials). Values set in both the
// shared config, and shared credentials will be taken from the shared
// credentials file. Enabling the Shared Config will also allow the Session
// to be built with retrieving credentials with AssumeRole set in the config.
//
// See the NewSessionWithOptions func for information on how to override or
// control through code how the Session will be created. Such as specifying the
// config profile, and controlling if shared config is enabled or not.
func New(cfgs ...*qiniu.Config) (*Session, error) {
	opts := Options{}
	opts.Config.MergeIn(cfgs...)

	return NewSessionWithOptions(opts)
}

// SharedConfigState provides the ability to optionally override the state
// of the session's creation based on the shared config being enabled or
// disabled.
type SharedConfigState int

// Options provides the means to control how a Session is created and what
// configuration values will be loaded.
type Options struct {
	// Provides config values for the SDK to use when creating service clients
	// and making API requests to services. Any value set in with this field
	// will override the associated value provided by the SDK defaults,
	// environment or config files where relevant.
	//
	// If not set, configuration values from from SDK defaults, environment,
	// config will be used.
	Config qiniu.Config

	// Ordered list of files the session will load configuration from.
	// It will override environment variable QINIU_SHARED_CREDENTIALS_FILE, QINIU_CONFIG_FILE.
	SharedConfigFiles []string

	// The handlers that the session and all API clients will be created with.
	// This must be a complete set of handlers. Use the defaults.Handlers()
	// function to initialize this value before changing the handlers to be
	// used by the SDK.
	Handlers request.Handlers
}

// NewSessionWithOptions returns a new Session created from SDK defaults, config files,
// environment, and user provided config files. This func uses the Options
// values to configure how the Session is created.
//
// If the QINIU_SDK_LOAD_CONFIG environment variable is set to a truthy value
// the shared config file (~/.qiniu/config) will also be loaded in addition to
// the shared credentials file (~/.qiniu/credentials). Values set in both the
// shared config, and shared credentials will be taken from the shared
// credentials file.
//     // Equivalent to session.New
//     sess := session.Must(session.NewSessionWithOptions(session.Options{}))
//
//     // Force enable Shared Config support
//     sess := session.Must(session.NewSessionWithOptions(session.Options{
//         SharedConfigState: session.SharedConfigEnable,
//     }))
func NewSessionWithOptions(opts Options) (*Session, error) {
	var envCfg envConfig

	envCfg = loadSharedEnvConfig()

	return newSession(opts, envCfg, &opts.Config)
}

// Must is a helper function to ensure the Session is valid and there was no
// error when calling a NewSession function.
//
// This helper is intended to be used in variable initialization to load the
// Session and configuration at startup. Such as:
//
//     var sess = session.Must(session.NewSession())
func Must(sess *Session, err error) *Session {
	if err != nil {
		panic(err)
	}

	return sess
}

func newSession(opts Options, envCfg envConfig, cfgs ...*qiniu.Config) (*Session, error) {
	cfg := defaults.Config()

	handlers := opts.Handlers
	if handlers.IsEmpty() {
		handlers = defaults.Handlers()
	}

	// Get a merged version of the user provided config to determine if
	// credentials were.
	userCfg := &qiniu.Config{}
	userCfg.MergeIn(cfgs...)

	// Ordered config files will be loaded in with later files overwriting
	// previous config file values.
	var cfgFiles []string
	if opts.SharedConfigFiles != nil {
		cfgFiles = opts.SharedConfigFiles
	} else {
		cfgFiles = []string{envCfg.SharedConfigFile, envCfg.SharedCredentialsFile}
	}
	// Load additional config from file(s)
	sharedCfg, err := loadSharedConfigDefaultSections(cfgFiles)
	if err != nil {
		return nil, err
	}

	if err := mergeConfigSrcs(cfg, userCfg, envCfg, sharedCfg, opts); err != nil {
		return nil, err
	}

	s := &Session{
		Config:   cfg,
		Handlers: handlers,
	}

	initHandlers(s)

	return s, nil
}

func mergeConfigSrcs(cfg, userCfg *qiniu.Config,
	envCfg envConfig, sharedCfg sharedConfig,
	sessOpts Options,
) error {
	// Merge in user provided configuration
	cfg.MergeIn(userCfg)

	// Configure credentials if not already set by the user when creating the
	// Session.
	if userCfg.Credentials == nil {
		creds, err := resolveCredentials(cfg, envCfg, sharedCfg, sessOpts)
		if err != nil {
			return err
		}
		cfg.Credentials = creds
	}
	mergeHostConfig(userCfg, cfg, envCfg, sharedCfg)

	return nil
}

// 合并来自用户配置的Host, 默认的HOST， 环境变量的Host, 和配置文件中的Host信息
// 优先级顺序用户代码中配置 > 环境变量配置 > 配置文件 > 默认配置
func mergeHostConfig(userCfg, defaultCfg *qiniu.Config, envCfg envConfig, sharedCfg sharedConfig) {
	defaultCfg.RsHost = mergeValue([]string{userCfg.RsHost, envCfg.RsHost, sharedCfg.RsHost, defaultCfg.RsHost})
	defaultCfg.RsfHost = mergeValue([]string{userCfg.RsfHost, envCfg.RsfHost, sharedCfg.RsfHost, defaultCfg.RsfHost})
	defaultCfg.UcHost = mergeValue([]string{userCfg.UcHost, envCfg.UcHost, sharedCfg.UcHost, defaultCfg.UcHost})
	defaultCfg.APIHost = mergeValue([]string{userCfg.APIHost, envCfg.APIHost, sharedCfg.APIHost, defaultCfg.APIHost})
}

func mergeValue(vs []string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

func initHandlers(s *Session) {
	// Add the Validate parameter handler if it is not disabled.
	s.Handlers.Validate.Remove(corehandlers.ValidateParametersHandler)
	if !qiniu.BoolValue(s.Config.DisableParamValidation) {
		s.Handlers.Validate.PushBackNamed(corehandlers.ValidateParametersHandler)
	}
}

// Copy creates and returns a copy of the current Session, coping the config
// and handlers. If any additional configs are provided they will be merged
// on top of the Session's copied config.
func (s *Session) Copy(cfgs ...*qiniu.Config) *Session {
	newSession := &Session{
		Config:   s.Config.Copy(cfgs...),
		Handlers: s.Handlers.Copy(),
	}

	initHandlers(newSession)

	return newSession
}

// ClientConfig satisfies the client.ConfigProvider interface and is used to
// configure the service client instances. Passing the Session to the service
// client's constructor (New) will use this method to configure the client.
func (s *Session) ClientConfig(cfgs ...*qiniu.Config) client.Config {
	// Backwards compatibility, the error will be eaten if user calls ClientConfig
	// directly. All SDK services will use ClientconfigWithError.
	cfg, _ := s.clientConfigWithErr(cfgs...)

	return cfg
}

func (s *Session) clientConfigWithErr(cfgs ...*qiniu.Config) (client.Config, error) {
	s = s.Copy(cfgs...)

	return client.Config{
		Config:   s.Config,
		Handlers: s.Handlers,
	}, nil
}
