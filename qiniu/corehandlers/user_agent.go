package corehandlers

import (
	"os"
	"runtime"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/request"
)

// SDKVersionUserAgentHandler is a request handler for adding the SDK Version
// to the user agent.
var SDKVersionUserAgentHandler = request.NamedHandler{
	Name: "core.SDKVersionUserAgentHandler",
	Fn: request.MakeAddToUserAgentHandler(qiniu.SDKName, qiniu.SDKVersion,
		runtime.Version(), runtime.GOOS, runtime.GOARCH),
}

const execEnvVar = `QINIU_EXECUTION_ENV`
const execEnvUAKey = `exec-env`

// AddHostExecEnvUserAgentHander is a request handler appending the SDK's
// execution environment to the user agent.
//
// If the environment variable QINIU_EXECUTION_ENV is set, its value will be
// appended to the user agent string.
var AddHostExecEnvUserAgentHander = request.NamedHandler{
	Name: "core.AddHostExecEnvUserAgentHander",
	Fn: func(r *request.Request) {
		v := os.Getenv(execEnvVar)
		if len(v) == 0 {
			return
		}

		request.AddToUserAgent(r, execEnvUAKey+"/"+v)
	},
}
