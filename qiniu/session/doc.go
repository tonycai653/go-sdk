/*
Package session 为SDK的服务客户端提供配置信息

如果多个服务客户端共享同样的配置， 那么一个Session实例可以在多个服务客户端之间共享.
Session 的配置信息和handlers会提供默认的配置， 用户在建立Session后可以修改配置.

最好可以把Session实例缓存起来， 因为Session的建立涉及到环境变量， 配置文件的加载.
每个信息的Session实例的建立都会重复该过程, 多个实例共享Session配置可以减少这个过程.

同步

只要没有goroutine修改该值，多个goroutine同步使用Session是安全的， 因为Session创建以后
SDK默认代码逻辑不会修改Session的信息.

Sessions from Shared Config

Sessions can be created using the method above that will only load the
additional config if the QINIU_SDK_LOAD_CONFIG environment variable is set.
Alternatively you can explicitly create a Session with shared config enabled.
To do this you can use NewSessionWithOptions to configure how the Session will
be created. Using the NewSessionWithOptions with SharedConfigState set to
SharedConfigEnable will create the session as if the QINIU_SDK_LOAD_CONFIG
environment variable was set.

创建Session对象

当创建Session对象的时候， 可以传入可选的api.Config配置信息来覆盖默认的配置信息.
这样可以针对特殊情况进行特殊化的配置.

By default NewSession will only load credentials from the shared credentials
file (~/.qiniu/credentials). If the QINIU_SDK_LOAD_CONFIG environment variable is
set to a truthy value the Session will be created from the configuration
values from the shared config (~/.qiniu/config) and shared credentials
(~/.qiniu/credentials) files. See the section Sessions from Shared Config for
more information.

Create a Session with the default config and request handlers. With credentials
and profile loaded from the environment and shared config automatically.
Requires the QINIU_PROFILE to be set, or "default" is used.

	// Create Session
	sess := session.Must(session.NewSession())

	// Create a kodo storage client instance from a session
	sess := session.Must(session.NewSession())

	svc := storage.New(sess)

Create Session With Option Overrides

In addition to NewSession, Sessions can be created using NewSessionWithOptions.
This func allows you to control and override how the Session will be created
through code instead of being driven by environment variables only.

Use NewSessionWithOptions when you want to provide the config profile, or
override the shared config state (QINIU_SDK_LOAD_CONFIG).

	// Equivalent to session.NewSession()
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		// Options
	}))

	// Specify profile to load for the session's config
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		 Profile: "profile_name",
	}))

	// Force enable Shared Config support
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

Adding Handlers

You can add handlers to a session for processing HTTP requests. All service
clients that use the session inherit the handlers. For example, the following
handler logs every request and its payload made by a service client:

	// Create a session, and add additional handlers for all service
	// clients created with the Session to inherit. Adds logging handler.
	sess := session.Must(session.NewSession())

	sess.Handlers.Send.PushFront(func(r *request.Request) {
		// Log every request made and its payload
		logger.Printf("Request: %s/%s, Payload: %s",
			r.ServiceName, r.Operation, r.Params)
	})

Shared Config Fields

By default the SDK will only load the shared credentials file's (~/.qiniu/credentials)
credentials values, and all other config is provided by the environment variables,
SDK defaults, and user provided api.Config values.

If the QINIU_SDK_LOAD_CONFIG environment variable is set, or SharedConfigEnable
option is used to create the Session the full shared config values will be
loaded. In addition the Session will load its configuration from both the shared config
file (~/.qiniu/config) and shared credentials file (~/.qiniu/credentials). Both
files have the same format.

If both config files are present the configuration from both files will be
read. The Session will be created from configuration values from the shared
credentials file (~/.qiniu/credentials) over those in the shared config file (~/.qiniu/config).

Credentials are the values the SDK should use for authenticating requests with
QINIU Services. They are from a configuration file will need to include both
qiniu_access_key_id and qiniu_secret_access_key must be provided together in the
same file to be considered valid. The values will be ignored if not a complete
group.

	qiniu_access_key_id = AKID
	qiniu_secret_access_key = SECRET

Environment Variables

When a Session is created several environment variables can be set to adjust
how the SDK functions, and what configuration data it loads when creating
Sessions. All environment values are optional, but some values like credentials
require multiple of the values to set or the partial values will be ignored.
All environment variable values are strings unless otherwise noted.

Environment configuration values. If set both Access Key ID and Secret Access
Key must be provided.

	# Access Key ID
	QINIU_ACCESS_KEY_ID=AKID
	QINIU_ACCESS_KEY=AKID # only read if QINIU_ACCESS_KEY_ID is not set.

	# Secret Access Key
	QINIU_SECRET_ACCESS_KEY=SECRET
	QINIU_SECRET_KEY=SECRET=SECRET # only read if QINIU_SECRET_ACCESS_KEY is not set.

Profile name the SDK should load use when loading shared config from the
configuration files. If not provided "default" will be used as the profile name.

	QINIU_PROFILE=my_profile

	# QINIU_DEFAULT_PROFILE is only read if QINIU_SDK_LOAD_CONFIG is also set,
	# and QINIU_PROFILE is not also set.
	QINIU_DEFAULT_PROFILE=my_profile

SDK load config instructs the SDK to load the shared config in addition to
shared credentials. This also expands the configuration loaded so the shared
credentials will have parity with the shared config file. This also enables
Profile support for the QINIU_DEFAULT_PROFILE env values as well.

	QINIU_SDK_LOAD_CONFIG=1

Shared credentials file path can be set to instruct the SDK to use an alternative
file for the shared credentials. If not set the file will be loaded from
$HOME/.qiniu/credentials on Linux/Unix based systems, and
%USERPROFILE%\.qiniu\credentials on Windows.

	QINIU_SHARED_CREDENTIALS_FILE=$HOME/my_shared_credentials

Shared config file path can be set to instruct the SDK to use an alternative
file for the shared config. If not set the file will be loaded from
$HOME/.qiniu/config on Linux/Unix based systems, and
%USERPROFILE%\.qiniu\config on Windows.

	QINIU_CONFIG_FILE=$HOME/my_shared_config
*/
package session
