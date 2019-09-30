package corehandlers

import "github.com/qiniu/go-sdk/qiniu/request"

// ValidateParametersHandler 校验输入参数的正确性
// 校验只有在请求发出去之前才有意义
var ValidateParametersHandler = request.NamedHandler{Name: "core.ValidateParametersHandler", Fn: func(r *request.Request) {
	if !r.ParamsFilled() {
		return
	}

	if v, ok := r.Params.(request.Validator); ok {
		if err := v.Validate(); err != nil {
			r.Error = err
		}
	}
}}
