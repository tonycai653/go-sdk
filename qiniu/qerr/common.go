package qerr

const (
	// ErrAuthorization -> httpStatusCode: 401, 认证授权失败。 包括密钥信息不正确；数字签名错误；授权已超时。
	ErrAuthorization = "AuthorizationError"

	// ErrParams -> httpStatusCode: 400, 请求报文格式错误。包括上传时，上传表单格式错误；URL 触发图片处理时，处理参数错误
	ErrParams = "ParamsError"

	// ErrPartFailed -> httpStatusCode: 298, 部分操作成功，部分失败
	// 一般是批处理接口batch会返回该错误
	ErrPartFailed = "PartError"

	// ErrAccessForbidden -> httpStatuscode: 403
	ErrAccessForbidden = "AccessDeniedError"

	// ErrNotFound -> httpStatusCode: 404, 资源不存在。 包括空间资源不存在；镜像源资源不存在。
	ErrNotFound = "NotFoundError"

	// ErrUnexpectedRequest -> httpStatusCode: 405, 请求方式错误。 主要指非预期的请求方式。
	ErrUnexpectedRequest = "UnexpectedRequestError"

	// ErrCrc32Verification -> httpStatusCode: 406, 上传的数据 CRC32 校验错误。
	ErrCrc32Verification = "Crc32VerificationError"

	// ErrAccountFrozen -> httpStatusCode: 419, 用户账号被冻结。
	ErrAccountFrozen = "AccountFrozenError"

	// ErrMirrorSourceRequest -> httpStatusCode: 478, 镜像回源失败。 主要指镜像源服务器出现异常。
	ErrMirrorSourceRequest = "MirrorSourceError"

	// ErrServiceUnavailable -> httpStatusCode: 503, 服务端不可用。
	ErrServiceUnavailable = "ServiceUnavailableError"

	// ErrServiceTimeout -> httpStatusCode: 504, 服务端操作超时。
	ErrServiceTimeout = "ServiceTimeoutError"

	// ErrRequestRate -> httpStatusCode: 573, 单个资源访问频率过高
	ErrRequestRate = "RequestRateError"

	// ErrUploadCallback -> httpStatusCode: 579, 上传成功但是回调失败。
	// 包括业务服务器异常；七牛服务器异常；服务器间网络异常。
	ErrUploadCallback = "UploadCallbackError"

	// ErrServiceOps -> httpStatusCode: 599, 服务端操作失败。
	ErrServiceOps = "SeriveOperationError"

	// ErrContentChanged -> httpStatusCode: 608, 资源内容被修改。
	ErrContentChanged = "ContentChangedError"

	// ErrResourceNotExist -> httpStatusCode: 612, 指定资源不存在或已被删除。
	ErrResourceNotExist = "ResourceNotExistError"

	// ErrResourceExist -> httpStatusCode: 614, 目标资源已存在。
	ErrResourceExist = "ResourceExistError"

	// ErrStorageLimit -> httpStatusCode: 630, 已创建的空间数量达到上限，无法创建新空间。
	ErrStorageLimit = "StorageNumberLimitError"

	// ErrStorageNotExist -> httpStatusCode: 631, 指定空间不存在。
	ErrStorageNotExist = "StorageNotExist"

	// ErrInvalidMarker -> httpStatusCode: 640, 调用列举资源 (list) 接口时，指定非法的marker参数。
	ErrInvalidMarker = "InvalidMarkerError"

	// ErrInvalidCtx -> httpStatusCode: 701, 在断点续上传过程中，后续上传接收地址不正确或ctx信息已过期。
	ErrInvalidCtx = "InvalidCtxError"

	// ErrConvertTypes 数据转换错误，比如string -> int
	ErrConvertTypes = "ConvertError"

	// ErrUnknown 未知错误
	ErrUnknown = "UnknownError"

	// ErrOpenFile 打开文件失败
	ErrOpenFile = "OpenFileError"

	// ErrStructFieldValidation 如果监测到不符合要求的字段，就会返回该错误
	// 有些函数或者方法对于输入的参数有要求， 比如不能是空， 不能为0等等
	ErrStructFieldValidation = "StructFieldError"

	// ErrCodeDeserialization is the deserialization error code that is received
	// during protocol unmarshaling.
	ErrCodeDeserialization = "DeserializationError"
)
