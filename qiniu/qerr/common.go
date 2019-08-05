package qerr

const (
	// 401 unathorized
	AuthorizationError = "AuthorizationError"

	// 400 Bad Request
	ParamsError = "ParamsError"

	// 数据解码错误
	DecodeError = "DecodeError"

	// 数据转换错误，比如string -> int
	ConvertError = "ConvertError"
)
