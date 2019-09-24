package credentials

// TokenType 定义了七牛签名算法的类型：
// QBoxToken, QiniuToken, BearToken, QiniuMacToken
type TokenType int

// 为保证用户数据的安全，七牛大部分 API HTTP请求都必须经过安全验证。
// 签名是七牛服务器用来识别用户身份与权限的凭证，我们采用 AK/SK(公钥/私钥)、token 来对用户进行身份验证。
// 根据签名算法的不同，可以分为Qiniu token, QBox token, BearToken, QiniuMacToken
// 对外公开的API最常用的是Qiniu token和QBox token
const (
	// 当请求的API接口不需要鉴权的时候，可以使用该类型
	TokenNone TokenType = iota

	// Qiniu token, 详细的算法参考：
	// https://developer.qiniu.com/kodo/kb/3702/QiniuToken
	TokenQiniu

	// QBox token, 又称管理凭证, 详细的算法请参考：
	// https://developer.qiniu.com/kodo/manual/1201/access-token
	TokenQBox

	TokenBear
	TokenQiniuMac
)
