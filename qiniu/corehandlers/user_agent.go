package corehandlers

import (
	"os"
	"runtime"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/request"
)

// SDKVersionUserAgentHandler 把SDK版本好加入到请求UserAgent中
var SDKVersionUserAgentHandler = request.NamedHandler{
	Name: "core.SDKVersionUserAgentHandler",
	Fn: request.MakeAddToUserAgentHandler(qiniu.SDKName, qiniu.SDKVersion,
		runtime.Version(), runtime.GOOS, runtime.GOARCH),
}

const execEnvVar = `QINIU_EXECUTION_ENV`
const execEnvUAKey = `exec-env`

// AddHostExecEnvUserAgentHander 把SDK执行环境加入到请求UserAgent中
//
// 如果环境变量 QINIU_EXECUTION_ENV 设置了, 该环境变量的值会被加入到请求UserAgent中
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
