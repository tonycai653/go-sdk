package session

import (
	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/client"
	"github.com/qiniu/go-sdk/qiniu/corehandlers"
	"github.com/qiniu/go-sdk/qiniu/defaults"
	"github.com/qiniu/go-sdk/qiniu/request"
)

const (
	// ErrCodeSharedConfig 代表处理文件配置信息时发生的错误
	ErrCodeSharedConfig = "SharedConfigErr"
)

// Session 是全局的总的配置入口, 多个客户端可以同时使用一个Session
// 满足client.ConfigProvider接口
type Session struct {
	Config   *qiniu.Config
	Handlers request.Handlers
}

// New 返回一个默认的配置session, 会从默认的环境变量，配置文件读取配置
// Session对象建立后，可以修改其中的配置项
//
// 配置文件(~/.qiniu/config) 会被读取以获取配置信息，密钥信息
func New(cfgs ...*qiniu.Config) (*Session, error) {
	opts := Options{}
	opts.Config.MergeIn(cfgs...)

	return NewSessionWithOptions(opts)
}

// Options 控制了Session如何创建， 创建后的配置信息内容
type Options struct {
	// 为SDK Session提供配置信息， 如果这个地方没有设置， SDK会使用从环境变量，配置文件
	// 加载的默认的配置信息， 否则会覆盖默认的配置
	Config qiniu.Config

	// 配置文件列表， 如果设置了， 会覆盖QINIU_CONFIG_FILE默认指定的配置文件
	SharedConfigFiles []string

	// 创建的Session对象的Handlers， 之后使用Session创建的服务客户端都会使用这套Handlers
	// 这个Handlers必须是完备的，完备指的是请求处理的各个阶段必需有合适的Handler, 比如请求阶段
	// 必须有一个请求Handler， 否则请求没法发出.
	//
	// 可以调用defaults.Handlers()函数生成这套完备的Handlers, 然后进行添加或者修改
	Handlers request.Handlers
}

// NewSessionWithOptions 创建一个Session, 返回Session指针
// 这个函数使用Options配置Session如何创建
//
//     // 和Session.New()等价
//     sess := session.Must(session.NewSessionWithOptions(session.Options{}))
func NewSessionWithOptions(opts Options) (*Session, error) {
	var envCfg envConfig

	envCfg = loadSharedEnvConfig()

	return newSession(opts, envCfg, &opts.Config)
}

// Must 是帮助函数，用来确定Session创建的过程没有发生错误， 如果发生了错误直接panic
//
// 这个帮助函数一般用来在Session创建的时候使用，比如:
// var sess = session.Must(session.NewSession())
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

	userCfg := &qiniu.Config{}
	userCfg.MergeIn(cfgs...)

	var cfgFiles []string
	if opts.SharedConfigFiles != nil {
		cfgFiles = opts.SharedConfigFiles
	} else {
		cfgFiles = []string{envCfg.SharedConfigFile}
	}
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
	cfg.MergeIn(userCfg)

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
	*defaultCfg.RsHost = mergeValue(userCfg.RsHost, &envCfg.RsHost, &sharedCfg.RsHost, defaultCfg.RsHost)
	*defaultCfg.RsfHost = mergeValue(userCfg.RsfHost, &envCfg.RsfHost, &sharedCfg.RsfHost, defaultCfg.RsfHost)
	*defaultCfg.UcHost = mergeValue(userCfg.UcHost, &envCfg.UcHost, &sharedCfg.UcHost, defaultCfg.UcHost)
	*defaultCfg.APIHost = mergeValue(userCfg.APIHost, &envCfg.APIHost, &sharedCfg.APIHost, defaultCfg.APIHost)
}

func mergeValue(vs ...*string) string {
	for _, v := range vs {
		if v != nil && *v != "" {
			return *v
		}
	}
	return ""
}

func initHandlers(s *Session) {
	// 把校验Request参数Params的handler加入到request.Handlers中，如果没有被禁用
	s.Handlers.Validate.Remove(corehandlers.ValidateParametersHandler)
	if !qiniu.BoolValue(s.Config.DisableParamValidation) {
		s.Handlers.Validate.PushBackNamed(corehandlers.ValidateParametersHandler)
	}
}

// Copy 复制当的Session, 返回一个新创建的Session
// 如果提供了额外的cfgs配置，这些配置信息会被合并到返回的Session中
func (s *Session) Copy(cfgs ...*qiniu.Config) *Session {
	newSession := &Session{
		Config:   s.Config.Copy(cfgs...),
		Handlers: s.Handlers.Copy(),
	}

	initHandlers(newSession)

	return newSession
}

// ClientConfig 实现了client.ConfigProvider接口
// 服务客户端会使用client.ConfigProvider接口来配置， 因此可以用Session对象来配置client
func (s *Session) ClientConfig(cfgs ...*qiniu.Config) client.Config {
	s = s.Copy(cfgs...)

	return client.Config{
		Config:   s.Config,
		Handlers: s.Handlers,
	}

}
