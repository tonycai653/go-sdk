// Package auth provides credential retrieval and management
//
// The Credentials is the primary method of getting access to and managing
// credentials Values. Using dependency injection retrieval of the credential
// values is handled by a object which satisfies the Provider interface.
//
// By default the Credentials.Get() will cache the successful result of a
// Provider's Retrieve() until Provider.IsExpired() returns true. At which
// point Credentials will call Provider's Retrieve() to get new credential Value.
//
// The Provider is responsible for determining when credentials Value have expired.
// It is also important to note that Credentials will always call Retrieve the
// first time Credentials.Get() is called.
package credentials

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	qhttp "github.com/qiniu/go-sdk/qiniu/http"
)

// AnonymousCredentials is an empty Credential object that can be used as
// dummy placeholder credentials for requests that do not need signed.
//
var AnonymousCredentials = NewStaticCredentials("", "")

// A Value is the QINIU credentials value for individual credential fields.
type Value struct {
	// QINIU Access key ID
	AccessKey string

	// QINIU Secret Access Key
	SecretKey []byte

	// Provider used to get credentials
	ProviderName string
}

// A Provider is the interface for any component which will provide credentials
// Value.
//
// The Provider should not need to implement its own mutexes, because
// that will be managed by Credentials.
type Provider interface {
	// Retrieve returns nil if it successfully retrieved the value.
	// Error is returned if the value were not obtainable, or empty.
	Retrieve() (Value, error)
}

// An ErrorProvider is a stub credentials provider that always returns an error
// this is used by the SDK when construction a known provider is not possible
// due to an error.
type ErrorProvider struct {
	// The error to be returned from Retrieve
	Err error

	// The provider name to set on the Retrieved returned Value
	ProviderName string
}

// Retrieve will always return the error that the ErrorProvider was created with.
func (p ErrorProvider) Retrieve() (Value, error) {
	return Value{ProviderName: p.ProviderName}, p.Err
}

// A Credentials provides concurrency safe retrieval of QINIU credentials Value.
// Credentials will cache the credentials value.
//
// Credentials is safe to use across multiple goroutines and will manage the
// synchronous state so the Providers do not need to implement their own
// synchronization.
//
// The first Credentials.Get() will always call Provider.Retrieve() to get the
// first instance of the credentials Value. All calls to Get() after that
// will return the cached credentials Value.
type Credentials struct {
	Value

	m sync.RWMutex

	provider Provider
}

// NewCredentials returns a pointer to a new Credentials with the provider set.
func NewCredentials(provider Provider) *Credentials {
	cred := &Credentials{
		provider: provider,
	}
	return cred
}

// Get returns the credentials value, or error if the credentials Value failed
// to be retrieved.
//
// Will return the cached credentials Value if the cache has cached not empty Value. If the
// credentials Value is not cached or the cached value is empty, the Provider's Retrieve() will be called
// to refresh the credentials.
func (c *Credentials) Get() (Value, error) {
	// Check the cached credentials first with just the read lock.
	c.m.RLock()
	defer c.m.RUnlock()

	if !c.Value.IsEmpty() {
		creds := c.Value
		c.m.RUnlock()
		return creds, nil
	} else {
		creds, err := c.provider.Retrieve()
		if err != nil {
			return Value{}, err
		}
		c.Value = creds
	}
	return c.Value, nil
}

// 构建一个Credentials对象指针
func New(accessKey, secretKey string) *Credentials {
	cred := NewStaticCredentials(accessKey, secretKey)
	return cred
}

// Sign 对数据进行签名，一般用于私有空间下载用途
func (v *Value) Sign(data []byte) (token string) {
	h := hmac.New(sha1.New, v.SecretKey)
	h.Write(data)

	sign := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s:%s", v.AccessKey, sign)
}

// SignWithData 对数据进行签名，一般用于上传凭证的生成用途
func (v *Value) SignWithData(b []byte) (token string) {
	encodedData := base64.URLEncoding.EncodeToString(b)
	sign := v.Sign([]byte(encodedData))
	return fmt.Sprintf("%s:%s", sign, encodedData)
}

func collectData(req *http.Request) (data []byte, err error) {
	u := req.URL
	s := u.Path
	if u.RawQuery != "" {
		s += "?"
		s += u.RawQuery
	}
	s += "\n"

	data = []byte(s)
	if incBody(req) {
		s2, rErr := bytesFromRequest(req)
		if rErr != nil {
			err = rErr
			return
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(s2))
		data = append(data, s2...)
	}
	return
}

func collectDataV2(req *http.Request) (data []byte, err error) {
	u := req.URL

	//write method path?query
	s := fmt.Sprintf("%s %s", req.Method, u.Path)
	if u.RawQuery != "" {
		s += "?"
		s += u.RawQuery
	}

	//write host and post
	s += "\nHost: "
	s += req.Host

	//write content type
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		s += "\n"
		s += fmt.Sprintf("Content-Type: %s", contentType)
	}
	s += "\n\n"

	data = []byte(s)
	//write body
	if incBodyV2(req) {
		s2, rErr := bytesFromRequest(req)
		if rErr != nil {
			err = rErr
			return
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(s2))
		data = append(data, s2...)
	}
	return
}

// SignRequest 对数据进行签名，一般用于管理凭证的生成
func (v *Value) SignRequest(req *http.Request) (token string, err error) {
	data, err := collectData(req)
	if err != nil {
		return
	}
	token = v.Sign(data)
	return
}

// SignRequestV2 对数据进行签名，一般用于高级管理凭证的生成
func (v *Value) SignRequestV2(req *http.Request) (token string, err error) {

	data, err := collectDataV2(req)
	if err != nil {
		return
	}
	token = v.Sign(data)
	return
}

// 管理凭证生成时，是否同时对request body进行签名
func incBody(req *http.Request) bool {
	return req.Body != nil && req.Body != http.NoBody && req.Header.Get("Content-Type") == qhttp.CONTENT_TYPE_FORM
}

func incBodyV2(req *http.Request) bool {
	contentType := req.Header.Get("Content-Type")
	return req.Body != nil && req.Body != http.NoBody && (contentType == qhttp.CONTENT_TYPE_FORM || contentType == qhttp.CONTENT_TYPE_JSON)
}

// VerifyCallback 验证上传回调请求是否来自七牛
func (v *Value) VerifyCallback(req *http.Request) (bool, error) {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return false, nil
	}

	token, err := v.SignRequest(req)
	if err != nil {
		return false, err
	}

	return auth == "QBox "+token, nil
}

// IsEmpty 返回密钥信息是否为空
// 当AccessKey 和SecretKey都是空的时候，返回true
// 否则返回false
func (v *Value) IsEmpty() bool {
	if len(v.AccessKey) <= 0 || len(v.SecretKey) <= 0 {
		return true
	}
	return false
}

// bytesFromRequest 读取http.Request.Body的内容到slice中， 返回[]byte
// 如果r.ContentLength < 0, 返回空的[]byte
// 如果r.ContentLength > 0, 返回读取的r.Body数据
// 如果r.ContentLength < 0 或者没有设置content-length头，直接返回读取的r.Body
// 读取数据过程中发生的错误，返回到err中
func bytesFromRequest(r *http.Request) (b []byte, err error) {
	if r.ContentLength == 0 {
		return
	}
	if r.ContentLength > 0 {
		b = make([]byte, int(r.ContentLength))
		_, err = io.ReadFull(r.Body, b)
		return
	}
	return ioutil.ReadAll(r.Body)
}
