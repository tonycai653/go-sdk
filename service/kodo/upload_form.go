package kodo

import (
	"io"
)

// FormInput 是表单上传的输入结构体
type FormInput struct {
	// 上传使用的域名，可以指定多个域名。
	// 每个域名可以有scheme, 比如http, https。
	// 如果没有指定scheme, 当UseHttps是`false`的时候，使用http上传，
	// 否则使用https上传。
	// 如果指定了scheme, 那么会忽略UseHttps的设置。
	UpHosts []string

	// 是否使用https上传
	UseHttps bool

	// 上传要保存在存储空间中的文件名
	Key string

	// 可选，用户自定义参数
	CustomParams map[string]string

	// 可选， 如果MimeType为空， 服务段会自动判断文件类型
	MimeType string

	// 自定义元数据，可同时自定义多个元数据。
	MetaKeys map[string]string

	// 当 HTTP 请求指定 accept 头部时，七牛会返回 Content-Type 头部值。
	// 该值用于兼容低版本 IE 浏览器行为。低版本 IE 浏览器在表单上传时，返回 application/json 表示下载
	// 返回 text/plain 才会显示返回内容。
	AcceptContentType string

	// 原文件名。对于没有文件名的情况，建议填入随机生成的纯文本字符串。本参数的值将作为魔法变量$(fname)的值使用。
	Filename string

	// 要上传的数据
	Data io.Reader
}

type FormOutput struct {
}

// UploadForm 使用表单上传的方式上传数据到七牛存储空间
// 数据大小小于等于defaults.DefaultsFormSize的时候，可以使用表单上传
// 大于该值的数据，建议使用分片上传
func (u *Kodo) UploadForm(input *FormInput, policy *PutPolicy) (out *FormOutput) {
}
