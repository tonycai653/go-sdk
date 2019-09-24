package kodo

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/qiniu/credentials"
	"github.com/qiniu/go-sdk/qiniu/qerr"
)

// PutPolicy 是七牛上传到存储空间的策略配置，可以配置诸如是否允许覆盖上传，
// 上传回调的地址， 上传返回内容的格式，允许上传的文件类型等等。
// 上传策略的详细文档： https://developer.qiniu.com/kodo/manual/1206/put-policy
//
// 推荐使用PutPolicy提供的方法设置字段， 会提供一些检查
type PutPolicy struct {
	Scope string `json:"scope"`

	// 不能超过uint32的最大值， 服务端bug
	Deadline        uint32 `json:"deadline"` // 截止时间（以秒为单位）
	IsPrefixalScope int    `json:"isPrefixalScope,omitempty"`
	InsertOnly      uint16 `json:"insertOnly,omitempty"` // 若非0, 即使Scope为 Bucket:Key 的形式也是insert only

	// 表示强制使用saveKey的值作为文件名， 不管用户是否指定文件名字
	// 当设置了ForceSaveKey为true时， 必须设置saveKey, 且saveKey的值不能为空

	//forceSaveKey=false，以客户端指定的 Key 为高优先级命名
	//    客户端已指定 Key，以 Key 命名
	//    客户端未指定 Key，上传策略中设置了 saveKey，以 saveKey 的格式命名。
	//    客户端未指定 Key，上传策略中未设置 saveKey，以文件 hash(etag) 命名。
	//forceSaveKey=true，以上传策略中的 saveKey 为高优先级命名；此时上传策略中的 saveKey 不允许为空
	//    客户端已指定 Key，以上传策略中 saveKey 的格式命名
	//    客户端未指定 Key，以上传策略中 saveKey 的格式命名
	ForceSaveKey bool `json:"forceSaveKey,omitempty"`

	DetectMime          uint8  `json:"detectMime,omitempty"` // 若非0, 则服务端根据内容自动确定 MimeType
	FsizeLimit          int64  `json:"fsizeLimit,omitempty"`
	FsizeMin            int64  `json:"fsizeMin,omitempty"`
	MimeLimit           string `json:"mimeLimit,omitempty"`
	SaveKey             string `json:"saveKey,omitempty"`
	CallbackFetchKey    uint8  `json:"callbackFetchKey,omitempty"`
	CallbackURL         string `json:"callbackUrl,omitempty"`
	CallbackHost        string `json:"callbackHost,omitempty"`
	CallbackBody        string `json:"callbackBody,omitempty"`
	CallbackBodyType    string `json:"callbackBodyType,omitempty"`
	ReturnURL           string `json:"returnUrl,omitempty"`
	ReturnBody          string `json:"returnBody,omitempty"`
	PersistentOps       string `json:"persistentOps,omitempty"`
	PersistentNotifyURL string `json:"persistentNotifyUrl,omitempty"`
	PersistentPipeline  string `json:"persistentPipeline,omitempty"`
	EndUser             string `json:"endUser,omitempty"`
	DeleteAfterDays     int    `json:"deleteAfterDays,omitempty"`
	FileType            int    `json:"fileType,omitempty"`
}

// NewPolicy 返回一个PutPolicy指针
// 返回的PutPolicy是一个空的结构体，需要调用WithXXX 系列方法设置字段
func NewPolicy() *PutPolicy {
	return &PutPolicy{}
}

// GetBucketName 解析p.Scope字段，返回解析出来的存储空间名字
func (p *PutPolicy) GetBucketName() (bucketName string) {
	scope := p.Scope
	if !strings.Contains(scope, ":") {
		bucketName = scope
	} else {
		splits := strings.SplitN(scope, ":", 2)
		bucketName = splits[0]
	}
	return
}

// WithScope 设置上传策略的scope字段, 这个字段是`必填的`。
// bucket 是存储空间的名字，不能为空
// key 是资源名称， 该字段的含义是变化的， 可以表示文件名， 文件前缀；
// 如果上传策略设置了IsPrefixalScope, 那么key表示文件前缀
//
// scope 字段的含义:
// 指定上传的目标资源空间 Bucket 和资源键 Key（最大为 750 字节）。有三种格式：
//
// <bucket>，表示允许用户上传文件到指定的 bucket。
// 在这种格式下文件只能新增（分片上传需要指定insertOnly为1才是新增，否则也为覆盖上传）,
// 若已存在同名资源（且文件内容/etag不一致），上传会失败；若已存在资源的内容/etag一致，则上传会返回成功。
//
// <bucket>:<key>，表示只允许用户上传指定 key 的文件。在这种格式下文件默认允许修改，若已存在同名资源则会被覆盖。
// 如果只希望上传指定 key 的文件，并且不允许修改，那么可以将下面的 insertOnly 属性值设为 1。
//
// <bucket>:<keyPrefix>，表示只允许用户上传指定以 keyPrefix 为前缀的文件，
// 当且仅当 isPrefixalScope 字段为 1 时生效，isPrefixalScope 为 1 时无法覆盖上传。
func (p *PutPolicy) WithScope(bucket, key string) *PutPolicy {
	p.Scope = bucket

	if key != "" {
		p.Scope = strings.Join([]string{p.Scope, key}, ":")
	}
	return p
}

// WithIsPrefixalScope 设置IsPrefixalScope字段, 该字段的含义如下：
//
// 若为 1，表示允许用户上传以 scope 的 keyPrefix 为前缀的文件。
//
// 如果isPrefixalScope参数为true, 设置p.IsprefixalScope为1， 否则为0
func (p *PutPolicy) WithIsPrefixalScope(isPrefixalScope bool) *PutPolicy {
	if isPrefixalScope {
		p.IsPrefixalScope = 1
	} else {
		p.IsPrefixalScope = 0
	}
	return p
}

// WithDeadline 设置p.Deadline字段
//
// 上传凭证有效截止时间, Unix时间戳，单位为秒。
// 该截止时间为上传完成后，在七牛空间生成文件的校验时间，而非上传的开始时间。
// 一般建议设置为上传开始时间 +3600s，用户可根据具体的业务场景对凭证截止时间进行调整。
func (p *PutPolicy) WithDeadline(deadline time.Time) *PutPolicy {
	p.Deadline = uint32(deadline.Unix())

	return p
}

// WithDeadlineAfter 设置p.Deadline过期时间为`current` + `d`
//
// 比如设置上传token在2019年8月20号后两小时后过期:
//
// p := &PutPolicy{}
// p.WithDeadlineAfter(time.Date(2019, 8, 20, 0, 0, 0, 0, nil), 2 * time.Hour)
func (p *PutPolicy) WithDeadlineAfter(current time.Time, d time.Duration) *PutPolicy {
	p.Deadline = uint32(current.Add(d).Unix())
	return p
}

// WithDeadlineAfterNow 设置p.Deadline过期时间为`d`时间后过期
//
// 比如设置上传token一小时后过期:
//
// p := &PutPolicy{}
// p.WithDeadlineAfterNow(1 * time.Hour)
func (p *PutPolicy) WithDeadlineAfterNow(d time.Duration) *PutPolicy {
	return p.WithDeadlineAfter(time.Now(), d)
}

// WithInsertOnly 设置InsertOnly字段
// 如果insertOnly 是`true`, 那么设置p.InsertOnly为1， 否则为0
//
//限定为新增语意。如果设置为非 0 值，则无论 scope 设置为什么形式，仅能以新增模式上传文件
func (p *PutPolicy) WithInsertOnly(insertOnly bool) *PutPolicy {
	if insertOnly {
		p.InsertOnly = 1
	} else {
		p.InsertOnly = 0
	}
	return p
}

// WithEndUser 设置p.EndUser字段
//
// 唯一属主标识。特殊场景下非常有用，例如根据 App-Client 标识给图片或视频打水印。
func (p *PutPolicy) WithEndUser(endUser string) *PutPolicy {
	p.EndUser = endUser
	return p
}

// WithReturnURL 设置p.ReturnURL字段
//
// Web 端文件上传成功后，浏览器执行 303 跳转的 URL。
// 通常用于表单上传, 文件上传成功后会跳转到 <returnUrl>?upload_ret=<queryString>，
// <queryString>包含 returnBody 内容。如不设置 returnUrl，则直接将 returnBody 的内容返回给客户端。
func (p *PutPolicy) WithReturnURL(returnURL string) *PutPolicy {
	p.ReturnURL = returnURL
	return p
}

// WithReturnBody 设置p.ReturnBody字段
//
// 上传成功后，自定义七牛云最终返回給上传端（在指定 returnUrl 时是携带在跳转路径参数中）的数据,支持魔法变量和自定义变量。
// returnBody 要求是合法的 JSON 文本。例如 {"key": $(key), "hash": $(etag), "w": $(imageInfo.width), "h": $(imageInfo.height)}。
// 关于魔法变量： https://developer.qiniu.com/kodo/manual/1235/vars#magicvar
// 关于自定义变量: https://developer.qiniu.com/kodo/manual/1235/vars#xvar
func (p *PutPolicy) WithReturnBody(returnBody string) *PutPolicy {
	p.ReturnBody = returnBody
	return p
}

// WithCallbackURL 设置p.CallbackURL上传回调地址
//
// 上传成功后，七牛云向业务服务器发送 POST 请求的 URL。
// 该URL必须是公网上可以正常进行 POST 请求并能响应 HTTP/1.1 200 OK 的有效 URL。
// 另外，为了给客户端有一致的体验，我们要求 callbackUrl 返回包 Content-Type 为 "application/json"，
// 即返回的内容必须是合法的 JSON 文本。
// 出于高可用的考虑，本字段允许设置多个 callbackUrl（用英文符号 ; 分隔），
// 在前一个 callbackUrl 请求失败的时候会依次重试下一个 callbackUrl。
//
// 一个典型例子是：http://<ip1>/callback;http://<ip2>/callback，并同时指定下面的 callbackHost 字段。
// 在 callbackUrl 中使用 ip 的好处是减少对 dns 解析的依赖，可改善回调的性能和稳定性。
// 指定 callbackUrl，必须指定 callbackbody，且值不能为空。
func (p *PutPolicy) WithCallbackURL(callbackUrls []string) *PutPolicy {
	p.CallbackURL = strings.Join(callbackUrls, ";")
	return p
}

// WithCallbackHost 设置p.CallbackHost字段
//
// 上传成功后，七牛云向业务服务器发送回调通知时的 Host 值。与 callbackUrl 配合使用，仅当设置了 callbackUrl 时才有效。
func (p *PutPolicy) WithCallbackHost(callbackHost string) *PutPolicy {
	p.CallbackHost = callbackHost
	return p
}

// WithCallbackBody 设置p.CallbackBody字段
//
// 上传成功后，七牛云向业务服务器发送 Content-Type: application/x-www-form-urlencoded 的 POST 请求。
// 业务服务器可以通过直接读取请求的 query 来获得该字段，支持魔法变量和自定义变量。
// callbackBody 要求是合法的 url query string。
// 例如key=$(key)&hash=$(etag)&w=$(imageInfo.width)&h=$(imageInfo.height)。
// 如果callbackBodyType指定为application/json，则callbackBody应为json格式，
// 例如:{"key":"$(key)","hash":"$(etag)","w":"$(imageInfo.width)","h":"$(imageInfo.height)"}。
func (p *PutPolicy) WithCallbackBody(callbackBody string) *PutPolicy {
	p.CallbackBody = callbackBody
	return p
}

// WithCallbackBodyType 设置p.CallbackBodyType字段
//
// 上传成功后，七牛云向业务服务器发送回调通知 callbackBody 的 Content-Type。
// 默认为 application/x-www-form-urlencoded，也可设置为 application/json。
func (p *PutPolicy) WithCallbackBodyType(callbackBodyType string) *PutPolicy {
	p.CallbackBodyType = callbackBodyType
	return p
}

// WithPersistentOps 设置p.PersistentOps字段
//
// 资源上传成功后触发执行的预转持久化处理指令列表。
// 支持魔法变量和自定义变量。
// 每个指令是一个 API 规格字符串，多个指令用;分隔。
// 请参阅persistenOps详解: https://developer.qiniu.com/kodo/manual/1206/put-policy#persistentOps
// 示例: https://developer.qiniu.com/kodo/manual/1206/put-policy#demo
// 同时添加 persistentPipeline 字段，使用专用队列处理，请参阅persistentPipeline:
// https://developer.qiniu.com/kodo/manual/1206/put-policy#put-policy-persistentPipeline
func (p *PutPolicy) WithPersistentOps(ops string) *PutPolicy {
	p.PersistentOps = ops
	return p
}

// WithPersistentNotifyURL 设置p.PersistentNotifyURL字段
//
// 接收持久化处理结果通知的 URL。
// 必须是公网上可以正常进行 POST 请求并能响应 HTTP/1.1 200 OK 的有效 URL。
// 该 URL 获取的内容和持久化处理状态查询的处理结果一致。
// 发送 body 格式是 Content-Type 为 application/json 的 POST 请求，需要按照读取流的形式读取请求的 body 才能获取。
func (p *PutPolicy) WithPersistentNotifyURL(notifyURL string) *PutPolicy {
	p.PersistentNotifyURL = notifyURL
	return p
}

// WithPersitentPipeline 设置p.PersistentPipeline字段
//
// 转码队列名。资源上传成功后，触发转码时指定独立的队列进行转码。
// 为空则表示使用公用队列，处理速度比较慢。建议使用专用队列
func (p *PutPolicy) WithPersitentPipeline(pipeline string) *PutPolicy {
	p.PersistentPipeline = pipeline
	return p
}

// WithForceSaveKey 设置p.ForceSaveKey字段
//
// saveKey的优先级设置。为 true 时，saveKey不能为空，会忽略客户端指定的key.
// 强制使用saveKey进行文件命名。参数不设置时，默认值为false
func (p *PutPolicy) WithForceSaveKey(forceSaveKey bool) *PutPolicy {
	p.ForceSaveKey = forceSaveKey
	return p
}

// WithSaveKey 设置p.SaveKey字段
//
// 自定义资源名, 支持魔法变量和自定义变量。
// forceSaveKey 为false时，这个字段仅当用户上传的时候没有主动指定 key 时起作用；
// forceSaveKey 为true时，将强制按这个字段的格式命名。
func (p *PutPolicy) WithSaveKey(saveKey string) *PutPolicy {
	p.SaveKey = saveKey
	return p
}

// WithFsizeMin 设置p.FsizeMin字段
//
// 限定上传文件大小最小值，单位Byte。
func (p *PutPolicy) WithFsizeMin(fsizeMin int64) *PutPolicy {
	p.FsizeMin = fsizeMin
	return p
}

// WithFsizeLimit 设置p.FsizeLimit字段
//
// 限定上传文件大小最大值，单位Byte。
// 超过限制上传文件大小的最大值会被判为上传失败，返回 413 状态码。
func (p *PutPolicy) WithFsizeLimit(fsizeLimit int64) *PutPolicy {
	p.FsizeLimit = fsizeLimit
	return p
}

// WithDetectMime 设置p.DetectMime字段
// 如果detectMime是`true`, 那么p.DetectMime = 1，否则为0
//
// 开启 MimeType 侦测功能, 设为非 0 值，则忽略上传端传递的文件 MimeType 信息，使用七牛服务器侦测内容后的判断结果。
// 默认设为 0 值，如上传端指定了 MimeType 则直接使用该值，否则按如下顺序侦测 MimeType 值：
// 1. 检查文件扩展名；
// 2. 检查 Key 扩展名；
// 3. 侦测内容。
// 如不能侦测出正确的值，会默认使用 application/octet-stream。
func (p *PutPolicy) WithDetectMime(detectMime bool) *PutPolicy {
	if detectMime {
		p.DetectMime = 1
	} else {
		p.DetectMime = 0
	}
	return p
}

// WithMimeLimit 设置p.MimeLimit字段
//
// 限定用户上传的文件类型, 指定本字段值，七牛服务器会侦测文件内容以判断 MimeType，
// 再用判断值跟指定值进行匹配，匹配成功则允许上传，匹配失败则返回 403 状态码。示例：
// image/*表示只允许上传图片类型
// image/jpeg;image/png表示只允许上传jpg和png类型的图片
// !application/json;text/plain表示禁止上传json文本和纯文本。注意最前面的感叹号！
func (p *PutPolicy) WithMimeLimit(mimeLimits []string) *PutPolicy {
	p.MimeLimit = strings.Join(mimeLimits, ";")
	return p
}

// WithFileType 设置p.FileType字段
//
// 文件存储类型。0 为普通存储（默认），1 为低频存储。
func (p *PutPolicy) WithFileType(fileType int) *PutPolicy {
	p.FileType = fileType
	return p
}

// UploadToken 通过上传策略生成上传凭证
// 上传策略的详细文档： https://developer.qiniu.com/kodo/manual/1206/put-policy
//
// Expires 字段对应于上传策略中的`deadline`字段，表示上传token过期的时间。
func (p *PutPolicy) UploadToken(cred *credentials.Credentials) (token string, err error) {
	v, gerr := cred.Get()
	if gerr != nil {
		err = qerr.New(credentials.ErrCredsRetrieve, "failed to get credentials value for UploadToken", err)
		return
	}
	// 默认一小时过期
	if p.Deadline <= 0 {
		p.WithDeadlineAfterNow(1 * time.Hour)
	}
	putPolicyJSON, _ := json.Marshal(p)
	token = v.SignWithData(putPolicyJSON)
	return
}
