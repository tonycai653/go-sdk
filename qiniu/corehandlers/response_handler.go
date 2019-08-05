package corehandlers

import (
	"encoding/json"

	"github.com/qiniu/go-sdk/qiniu/qerr"
	"github.com/qiniu/go-sdk/qiniu/request"
)

// UnmarshalHandler 反序列化http.Body到相应的结构体中
var UnmarshalHandler = request.NamedHandler{
	Name: "UnmarshalHandler",
	Fn: func(r *request.Request) {
		if r.DataFilled() {
			contentType := r.HTTPResponse.Header.Get("Content-Type")

			switch contentType {
			case "application/json":
				err := json.NewDecoder(r.HTTPResponse.Body).Decode(r.Data)
				if err != nil {
					r.Error = qerr.New(request.ErrCodeDeserialization, "failed to decode data with content-type: "+contentType, err)
					return
				}
			}
		}
	},
}
