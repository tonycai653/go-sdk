package request

import (
	"fmt"
	"strings"
)

// Handlers 是发出网络请求和响应处理的的各个阶段的对Request的处理函数
// 每个阶段可以有多个处理函数
type Handlers struct {
	Validate         HandlerList
	Build            HandlerList
	Sign             HandlerList
	Send             HandlerList
	ValidateResponse HandlerList
	Unmarshal        HandlerList
	UnmarshalStream  HandlerList
	UnmarshalMeta    HandlerList
	UnmarshalError   HandlerList
	Retry            HandlerList
	AfterRetry       HandlerList
	CompleteAttempt  HandlerList
	Complete         HandlerList
}

// Copy 返回一份Handlers拷贝
func (h *Handlers) Copy() Handlers {
	return Handlers{
		Validate:         h.Validate.copy(),
		Build:            h.Build.copy(),
		Sign:             h.Sign.copy(),
		Send:             h.Send.copy(),
		ValidateResponse: h.ValidateResponse.copy(),
		Unmarshal:        h.Unmarshal.copy(),
		UnmarshalStream:  h.UnmarshalStream.copy(),
		UnmarshalError:   h.UnmarshalError.copy(),
		UnmarshalMeta:    h.UnmarshalMeta.copy(),
		Retry:            h.Retry.copy(),
		AfterRetry:       h.AfterRetry.copy(),
		CompleteAttempt:  h.CompleteAttempt.copy(),
		Complete:         h.Complete.copy(),
	}
}

// Clear 清楚所有的handler
func (h *Handlers) Clear() {
	h.Validate.Clear()
	h.Build.Clear()
	h.Send.Clear()
	h.Sign.Clear()
	h.Unmarshal.Clear()
	h.UnmarshalStream.Clear()
	h.UnmarshalMeta.Clear()
	h.UnmarshalError.Clear()
	h.ValidateResponse.Clear()
	h.Retry.Clear()
	h.AfterRetry.Clear()
	h.CompleteAttempt.Clear()
	h.Complete.Clear()
}

// IsEmpty 返回是否handler为空
// 只有当所有阶段的handler列表为空， 才返回空
func (h *Handlers) IsEmpty() bool {
	if h.Validate.Len() != 0 {
		return false
	}
	if h.Build.Len() != 0 {
		return false
	}
	if h.Send.Len() != 0 {
		return false
	}
	if h.Sign.Len() != 0 {
		return false
	}
	if h.Unmarshal.Len() != 0 {
		return false
	}
	if h.UnmarshalStream.Len() != 0 {
		return false
	}
	if h.UnmarshalMeta.Len() != 0 {
		return false
	}
	if h.UnmarshalError.Len() != 0 {
		return false
	}
	if h.ValidateResponse.Len() != 0 {
		return false
	}
	if h.Retry.Len() != 0 {
		return false
	}
	if h.AfterRetry.Len() != 0 {
		return false
	}
	if h.CompleteAttempt.Len() != 0 {
		return false
	}
	if h.Complete.Len() != 0 {
		return false
	}

	return true
}

// HandlerListRunItem 代表Handler列表中的列表项
type HandlerListRunItem struct {
	Index   int
	Handler NamedHandler
	Request *Request
}

// HandlerList 管理一个Handler队列
type HandlerList struct {
	list []NamedHandler

	// 该函数在handler列表中的每个handler被调用后调用， 如果该函数返回true, 那么会接着处理下一个handler
	// 否则， 停止处理handler列表
	AfterEachFn func(item HandlerListRunItem) bool
}

// NamedHandler 包含一个handler名字，和执行函数
type NamedHandler struct {
	Name string
	Fn   func(*Request)
}

func (l *HandlerList) copy() HandlerList {
	n := HandlerList{
		AfterEachFn: l.AfterEachFn,
	}
	if len(l.list) == 0 {
		return n
	}

	n.list = append(make([]NamedHandler, 0, len(l.list)), l.list...)
	return n
}

// Clear 清理handler队列为空
func (l *HandlerList) Clear() {
	l.list = l.list[0:0]
}

// Len 返回handler队列的长度
func (l *HandlerList) Len() int {
	return len(l.list)
}

// PushBack 把匿名handler f放到队列尾部
func (l *HandlerList) PushBack(f func(*Request)) {
	l.PushBackNamed(NamedHandler{"__anonymous", f})
}

// PushBackNamed 把NamedHandler n 放到队列尾部
func (l *HandlerList) PushBackNamed(n NamedHandler) {
	if cap(l.list) == 0 {
		l.list = make([]NamedHandler, 0, 5)
	}
	l.list = append(l.list, n)
}

// PushFront 把匿名handler f 放到队列头部
func (l *HandlerList) PushFront(f func(*Request)) {
	l.PushFrontNamed(NamedHandler{"__anonymous", f})
}

// PushFrontNamed 把NamedHandler n 放到队列头部
func (l *HandlerList) PushFrontNamed(n NamedHandler) {
	if cap(l.list) == len(l.list) {
		// Allocating new list required
		l.list = append([]NamedHandler{n}, l.list...)
	} else {
		// Enough room to prepend into list.
		l.list = append(l.list, NamedHandler{})
		copy(l.list[1:], l.list)
		l.list[0] = n
	}
}

// Remove 把n从队列中删除
func (l *HandlerList) Remove(n NamedHandler) {
	l.RemoveByName(n.Name)
}

// RemoveByName 从队列中找到名字为name的handler, 删除该handler
func (l *HandlerList) RemoveByName(name string) {
	for i := 0; i < len(l.list); i++ {
		m := l.list[i]
		if m.Name == name {
			copy(l.list[i:], l.list[i+1:])
			l.list[len(l.list)-1] = NamedHandler{}
			l.list = l.list[:len(l.list)-1]
			i--
		}
	}
}

// SwapNamed 从队列中找到名字和n一样的handler,并用n替换掉该handler
// 如果发生了替换，返回true, 否则false
func (l *HandlerList) SwapNamed(n NamedHandler) (swapped bool) {
	for i := 0; i < len(l.list); i++ {
		if l.list[i].Name == n.Name {
			l.list[i].Fn = n.Fn
			swapped = true
		}
	}

	return swapped
}

// Swap 从队列中找到名字为name的handler, 并用replace替换
// 如果发生了替换返回true, 否则false
func (l *HandlerList) Swap(name string, replace NamedHandler) bool {
	var swapped bool

	for i := 0; i < len(l.list); i++ {
		if l.list[i].Name == name {
			l.list[i] = replace
			swapped = true
		}
	}

	return swapped
}

// SetBackNamed will replace the named handler if it exists in the handler list.
// If the handler does not exist the handler will be added to the end of the list.
func (l *HandlerList) SetBackNamed(n NamedHandler) {
	if !l.SwapNamed(n) {
		l.PushBackNamed(n)
	}
}

// SetFrontNamed will replace the named handler if it exists in the handler list.
// If the handler does not exist the handler will be added to the beginning of
// the list.
func (l *HandlerList) SetFrontNamed(n NamedHandler) {
	if !l.SwapNamed(n) {
		l.PushFrontNamed(n)
	}
}

// Run 执行队列中的所有handler
func (l *HandlerList) Run(r *Request) {
	for i, h := range l.list {
		h.Fn(r)
		item := HandlerListRunItem{
			Index: i, Handler: h, Request: r,
		}
		if l.AfterEachFn != nil && !l.AfterEachFn(item) {
			return
		}
	}
}

// HandlerListLogItem 打印handler日志信息， 并且总是返回true, 以继续处理handler队列
func HandlerListLogItem(item HandlerListRunItem) bool {
	if item.Request.Config.Logger == nil {
		return true
	}
	item.Request.Config.Logger.Log("DEBUG: RequestHandler",
		item.Index, item.Handler.Name, item.Request.Error)

	return true
}

// HandlerListStopOnError 当request.Error不为nil的时候返回true, 停止处理request handler队列
// 否则返回false, 继续处理
func HandlerListStopOnError(item HandlerListRunItem) bool {
	return item.Request.Error == nil
}

// WithAppendUserAgent 把s加入到当前的UserAgent后面， 以空格分割
func WithAppendUserAgent(s string) Option {
	return func(r *Request) {
		r.Handlers.Build.PushBack(func(r2 *Request) {
			AddToUserAgent(r, s)
		})
	}
}

// MakeAddToUserAgentHandler 把 name/version 格式的字符串加入到请求User-Agent头中.
// 如果参数extra 非空， 这个参数也会被加入到请求User-Agent中， 形成如下的格式:
// "name/version (extra0; extra1; ...)"
func MakeAddToUserAgentHandler(name, version string, extra ...string) func(*Request) {
	ua := fmt.Sprintf("%s/%s", name, version)
	if len(extra) > 0 {
		ua += fmt.Sprintf(" (%s)", strings.Join(extra, "; "))
	}
	return func(r *Request) {
		AddToUserAgent(r, ua)
	}
}

// MakeAddToUserAgentFreeFormHandler 把字符串s加入到UserAgent中
func MakeAddToUserAgentFreeFormHandler(s string) func(*Request) {
	return func(r *Request) {
		AddToUserAgent(r, s)
	}
}
