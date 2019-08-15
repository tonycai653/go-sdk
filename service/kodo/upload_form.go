package kodo

import (
	"bytes"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/qerr"
	"github.com/qiniu/go-sdk/qiniu/request"
)

const (
	// ErrWriteField 表单上传， 格式化表单的字段的时候报错
	ErrWriteField = "WriteFieldError"

	// ErrQueryDomains 请求kodo/v3query接口出错
	ErrQueryDomains = "QueryDomainsError"

	// ErrNoUpHosts 没有找到上传host
	ErrNoUpHosts = "NoUpHostsError"

	// ErrUpTokenEmpty 上传token是空
	ErrUpTokenEmpty = "UpTokenEmptyError"
)

// FormInput 是表单上传的输入结构体, 其中UpToken字段是必须填写的
// 如果指定了Filename字段， 那么会使用该文件的数据作为Data的内容。
// 如果没有指定Filename, 那么Data字段必须设置
// Filename字段和Data字段， 两个必填其一
type FormInput struct {
	// 上传使用的域名，可以指定多个域名。
	// 每个域名可以有scheme, 比如http, https。
	// 如果没有指定scheme, 当UseHttps是`false`的时候，使用http上传， 否则使用https上传。
	// 如果指定了scheme, 那么会忽略UseHttps的设置。
	UpHosts []string

	// 上传域名选择器
	Selector HostsSelector

	// 是否使用https上传
	UseHTTPS bool

	// 上传要保存在存储空间中的文件名, 必须是UTF-8编码。
	// 如果上传凭证中 scope 指定为 <bucket>:<key>， 则该字段也必须指定，并且与上传凭证中的 key 一致，否则会报403错误。
	// 如果表单没有指定 key，可以使用上传策略saveKey字段所指定魔法变量生成 Key，如果没有模板，则使用 Hash 值作为 Key。
	Key string

	// 要上传的存储空间名字, 如果UpToken和PutPolicy都没有设置， 那么使用该名字设置PutPlicy scope字段.
	// 如果同时设置了BucketName和PutPolicy， 那么最终上传的bucket是PutPolicy中的scope指定的存储空间名字， 该字段被忽略.
	BucketName string

	// 要上传的存储空间所在区域
	Region string

	// 是否开启Crc32校验
	// WithCrc32 是`true`就开启校验，否则不开启
	WithCrc32 bool

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
	OrigFilename string

	// 要上传的数据
	// 如果该值为nil, 那么上传的文件内容为空
	Data io.Reader

	// 要上传的文件名字, 不要同时设置该字段和Data字段
	// 如果两个字段同时设置了，那么会使用Data字段的内容上传
	Filename string

	// 上传策略， 如果UpToken为空， 并且该上传策略设置了内容，那么使用该字段计算上传token
	PutPolicy *PutPolicy

	// 上传token
	// 如果该字段非空， 那么忽略PutPolicy字段和BucketName字段
	UpToken string

	// 调用者不需要设置该字段
	kodo *Kodo
}

func (input *FormInput) init(u *Kodo) error {
	if input.kodo == nil {
		input.kodo = u
	}
	if err := input.validateFields(); err != nil {
		return err
	}
	if err := input.setData(); err != nil {
		return err
	}
	return nil
}

func (input *FormInput) validateFields() error {
	if input.Key == "" {
		return qerr.New(qerr.ErrStructFieldValidation, "FormInput.Key field cannot be empty", nil)
	}
	if input.UpToken == "" {
		if input.PutPolicy == nil && input.BucketName != "" {
			input.PutPolicy = &PutPolicy{
				Scope: input.BucketName,
			}
		}
		if input.PutPolicy == nil {
			return qerr.New(ErrUpTokenEmpty, "upload token cannot be empty", nil)
		}
		upToken, err := input.PutPolicy.UploadToken(input.kodo.Config.Credentials)
		if err != nil {
			return err
		}
		input.UpToken = upToken
	}
	_, scope, err := DecodeUpToken(input.UpToken)
	if err != nil {
		return qerr.New(qerr.ErrCodeDeserialization, "failed to decode upload token: "+input.UpToken, err)
	}
	if !strings.Contains(scope, ":") {
		input.BucketName = scope
	} else {
		splits := strings.SplitN(scope, ":", 2)
		input.BucketName = splits[0]
	}
	return nil
}

func (input *FormInput) getOrigFilename() string {
	if input.OrigFilename != "" {
		return input.OrigFilename
	}
	return randomString(10)
}

func (input *FormInput) setSelector() error {
	if input.Selector != nil {
		return nil
	}
	if len(input.UpHosts) > 0 {
		input.Selector = NewSelector(input.UpHosts)
		return nil
	}
	var rd RegionDomains
	if input.Region != "" {
		region := GetDefaultRegion(input.Region)
		rd = region.RegionDomains
	}
	if rd.AllUpDomainGroupEmpty() {
		regionDomains, err := input.kodo.QueryRegionDomains(input.BucketName)
		if err != nil {
			return qerr.New(ErrQueryDomains, fmt.Sprintf("query region domains error for bucket: %s", input.BucketName), err)
		}
		rd = *regionDomains
	}
	grp := rd.SelectUpDomainGroup()
	if grp.IsEmpty() {
		return qerr.New(ErrNoUpHosts, "no upload host found", nil)
	}
	if len(grp.Main) > 0 {
		input.Selector = NewSelector(grp.Main)
	} else {
		input.Selector = NewSelector(grp.Backup)
	}
	return nil
}

func (input *FormInput) setData() error {
	if input.Filename != "" {
		file, err := os.Open(input.Filename)
		if err != nil {
			return qerr.New(qerr.ErrOpenFile, fmt.Sprintf("failed to open file `%s`", input.Filename), err)
		}
		input.Data = file
	}
	// Region配置来自input和初始化session的时候的配置， 优先使用input中的配置
	if input.kodo.Config.Region != "" && input.Region == "" {
		input.Region = input.kodo.Config.Region
	}
	if err := input.setSelector(); err != nil {
		return err
	}
	return nil
}

func (input *FormInput) getUpHost() (host, scheme string, err error) {
	h := input.Selector.Select()
	host, scheme, err = qiniu.NormalizeHost(h)
	if err != nil {
		return
	}
	if input.UseHTTPS {
		scheme = "https"
	} else {
		if strings.TrimSpace(scheme) == "" {
			scheme = "http"
		}
	}
	return host, scheme, nil
}

// DefaultFormOutput 是表单上传接口的数据返回数据承载体
// 该结构体只定义了默认情况下的返回值， 如果上传策略中定义了returnBody，
// 服务端返回的数据会有其他的字段， 需要调用者定义相应的结构体.
type DefaultFormOutput struct {
	// 上传文件的hash值
	Hash string `json:"hash,omitempty"`

	// 上传文件保存在存储中的文件名
	Key string `json:"key,omitempty"`
}

// UploadForm 使用表单上传的方式上传数据到七牛存储空间
// 数据大小小于等于defaults.DefaultsFormSize的时候，可以使用表单上传.
// 大于该值的数据，建议使用分片上传, 如果使用表单上传的文件太大可能会造成内存溢出。
// 如果上传策略中定义了returnBody, 那么接口返回的数据可能不只hash和key, 还有其他的内容。
// 这个时候调用者要根据上传策略定义适合的结构体，作为out参数传递给该方法。
// 示例: 该例子上传到存储空间`test`, 保存在空间的名字为`key.txt`
// session := session.New()
// kodoClient := kodo.New(session)
// formInput := &kodo.FormInput{
//      BucketName: "test",
//      Key: "key.txt",
//      Data: strings.NewReader("hello world")
// }
// out := DefaultFormOutput{}
// err := kodoClient.UploadForm(formInput, &out)
// if err != nil {
//     fmt.Println(err)
//     os.Exit(1)
// }
// fmt.Println(out)
func (u *Kodo) UploadForm(input *FormInput, out interface{}) error {
	req, err := u.UploadFormRequest(input, out)
	if err != nil {
		return err
	}
	return req.Send()
}

// UploadFormRequest 返回request.Request指针， 用于发起表单上传请求
// 同时，返回FormOutput结构指针。
func (u *Kodo) UploadFormRequest(input *FormInput, out interface{}) (req *request.Request, err error) {

	// do some setup work, set fields and sanity check
	if e := input.init(u); e != nil {
		err = e
		return
	}

	var data io.Reader

	var h hash.Hash32
	if input.WithCrc32 {
		h = crc32.NewIEEE()
		data = io.TeeReader(input.Data, h)
	} else {
		data = input.Data
	}

	var b bytes.Buffer

	writer := &multipartWriter{
		Writer: multipart.NewWriter(&b),
	}
	// write custom names
	for cstName, v := range input.CustomParams {
		cstName = customName(cstName)
		if e := writer.WriteField(customName(cstName), v); e != nil {
			err = qerr.New(ErrWriteField, fmt.Sprintf("failed to write form `%s` field", cstName), e)
			return
		}
	}

	// write x-qn-meta keys
	for cstName, v := range input.MetaKeys {
		cstName = metaName(cstName)
		if e := writer.WriteField(cstName, v); e != nil {
			err = qerr.New(ErrWriteField, fmt.Sprintf("failed to write form `%s` field", cstName), e)
			return
		}
	}

	if e := writer.WriteField("key", input.Key); e != nil {
		err = qerr.New(ErrWriteField, "failed to write form `key` field", e)
		return
	}
	if e := writer.WriteField("token", input.UpToken); e != nil {
		err = qerr.New(ErrWriteField, "failed to write form `token` field", e)
		return
	}
	if e := writer.WriteField("accept", input.AcceptContentType); e != nil {
		err = qerr.New(ErrWriteField, "failed to write form `accept` field", e)
		return
	}
	if e := writer.writeFormFileField("file", input.getOrigFilename(), input.MimeType, data); e != nil {
		err = qerr.New(ErrWriteField, "failed to write form `file` field", e)
		return
	}
	// after writeFormFileField to make sure that file data read to hash
	if input.WithCrc32 {
		if e := writer.WriteField("crc32", fmt.Sprintf("%010d", h.Sum32())); e != nil {
			err = qerr.New(ErrWriteField, "failed to write form `crc32` field", e)
			return
		}
	}
	if e := writer.Close(); e != nil {
		err = qerr.New(ErrWriteField, "failed to close multipart writer", e)
		return
	}
	host, scheme, nerr := input.getUpHost()
	if nerr != nil {
		err = nerr
		return
	}
	op := &request.API{
		Scheme:      scheme,
		Path:        "/",
		Method:      "POST",
		Host:        host,
		ContentType: writer.FormDataContentType(),
		APIName:     "form-upload",
		ServiceName: ServiceName,
	}
	req = u.newRequest(op, bytes.NewReader(b.Bytes()), out)
	return
}

// 从mime/multipart官方库拷贝而来
// 官方库CreateFormFile不支持设置文件类型， 为了支持调用者设置文件类型， 拷贝过来从新实现该方法
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

type multipartWriter struct {
	*multipart.Writer
}

func (w *multipartWriter) WriteField(fieldname, value string) error {
	if value != "" {
		return w.Writer.WriteField(fieldname, value)
	}
	return nil
}

func (w *multipartWriter) createFormFile(fieldname, filename, contentType string) (io.Writer, error) {

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(fieldname), escapeQuotes(filename)))
	if contentType == "" {
		h.Set("Content-Type", "application/octet-stream")
	} else {
		h.Set("Content-Type", contentType)
	}
	return w.CreatePart(h)
}

func (w *multipartWriter) writeFormFileField(fieldname, filename, contentType string, data io.Reader) error {
	bs, err := ioutil.ReadAll(data)
	if err != nil {
		return err
	}
	p, cerr := w.createFormFile(fieldname, filename, contentType)
	if cerr != nil {
		return cerr
	}
	_, werr := p.Write(bs)
	if werr != nil {
		return werr
	}
	return nil
}

var letters = "abcdefghijklmnopqrstuvwxyz"

func randomString(n int) string {
	bs := make([]byte, n)
	for i := 0; i < n; i++ {
		bs[i] = letters[rand.Intn(len(letters))]
	}
	return string(bs)
}

func customName(name string) string {
	if strings.HasPrefix(name, "x:") {
		return name
	}
	return "x:" + name
}

func metaName(name string) string {
	if strings.HasPrefix(name, "x-qn-meta-") {
		return name
	}
	return "x-qn-meta-" + name
}
