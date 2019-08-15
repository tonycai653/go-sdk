package kodo

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// FileInfo 文件基本信息
type FileInfo struct {
	Hash     string `json:"hash"`
	Fsize    int64  `json:"fsize"`
	PutTime  int64  `json:"putTime"`
	MimeType string `json:"mimeType"`
	Type     int    `json:"type"`
}

// String 返回表示文件信息的字符串
func (f *FileInfo) String() string {
	str := ""
	str += fmt.Sprintf("Hash:     %s\n", f.Hash)
	str += fmt.Sprintf("Fsize:    %d\n", f.Fsize)
	str += fmt.Sprintf("PutTime:  %d\n", f.PutTime)
	str += fmt.Sprintf("MimeType: %s\n", f.MimeType)
	str += fmt.Sprintf("Type:     %d\n", f.Type)
	return str
}

// HostsSelector 定义了一组上传域名的选择策略
// 比如每次只选择固定index的域名
// 也可以轮流选择域名， 每次选择的域名确保和上次不一样
type HostsSelector interface {
	// 从一组域名中选择一个域名
	Select() string
}

// ErrorUpdator 抽象使用域名出错， 跟新错误信息
type ErrorUpdator interface {

	// 跟新host使用出错的信息
	Update(host string, err error)
}

// HostsUpdatorSelector 是HostsSelector和ErrorUpdator的合成
type HostsUpdatorSelector interface {
	ErrorUpdator
	HostsSelector
}

type errTime struct {
	err error
	t   time.Time
}

func (e errTime) Error() string {
	return fmt.Sprintf("%s %s", e.t.String(), e.err.Error())
}

type host struct {
	ht string
	// 按照errTime.t时间从小到大排序
	// 域名的使用错误是天然的按照时间从小到大的
	errs []errTime
	mu   sync.Mutex
}

func (h *host) setupError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	e := errTime{
		err: err,
		t:   time.Now(),
	}
	h.errs = append(h.errs, e)
	// 只保留最近的10个错误
	if len(h.errs) > 10 {
		h.errs = h.errs[1:]
	}
}

func (h *host) firstError() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.errs) > 0 {
		return h.errs[0]
	}
	return nil
}

func (h *host) lastError() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.errs) > 0 {
		return h.errs[len(h.errs)-1]
	}
	return nil
}

func (h *host) errFrequency() float64 {
	if len(h.errs) <= 0 { // no error
		return 0.0
	}
	// firstError 和 lastError都持有锁
	min := h.firstError().(errTime)
	max := h.lastError().(errTime)

	return float64(len(h.errs)) / float64(max.t.Sub(min.t)/time.Second)
}

// Selector 从一组域名中选择单位时间内发生错误数最少的域名
// Selector 只能从一组域名中选择域名， 如果域名组发生变化，需要重新创建Selector对象
type Selector struct {
	hosts map[string]*host
}

// GroupSelector 在主和从域名组选择一个域名
type GroupSelector struct {
	main   map[string]*host
	backup map[string]*host
}

// NewSelector 返回Selector指针， 是默认的上传选择器
func NewSelector(hs []string) *Selector {
	s := &Selector{
		hosts: make(map[string]*host),
	}
	for _, h := range hs {
		s.hosts[h] = &host{
			ht: h,
		}
	}
	return s
}

// Select 从一组域名中选择错误率做少的域名
func (s *Selector) Select() string {
	var minfreq = math.MaxFloat64
	var th *host

	for _, h := range s.hosts {
		freq := h.errFrequency()
		if freq < minfreq {
			minfreq = freq
			th = h
		}
	}
	return th.ht
}

// Update 更新域名使用中的错误
func (s *Selector) Update(host string, err error) {
	if _, ok := s.hosts[host]; ok {
		s.hosts[host].setupError(err)
	}
}

// FixedSelector 始终从一组列表中选择第一个域名
type FixedSelector struct {
	hosts []string
}

// NewFixedSelector 返回FixedSelector指针
func NewFixedSelector(hosts []string) *FixedSelector {
	s := &FixedSelector{
		hosts: hosts,
	}
	return s
}

// Select 返回列表中第一个域名
// 如果域名列表为空， 返回空字符串
func (s *FixedSelector) Select() string {
	if len(s.hosts) > 0 {
		return s.hosts[0]
	}
	return ""
}

// RoundRobinSelector 实现HostsSelector接口
// RoundRobinSelector 是线程安全的，可以有多个线程同时调用Select函数
type RoundRobinSelector struct {
	hosts []string

	mu        sync.Mutex
	lastIndex int
}

// NewRoundRobinSelector 返回一个RoundRobinSelector指针
func NewRoundRobinSelector(hosts []string) *RoundRobinSelector {
	selector := &RoundRobinSelector{
		hosts:     hosts,
		lastIndex: -1,
	}
	return selector
}

// Select 从一组上传域名中选择一个上传域名
// 如果所有域名都已经选择完毕， 在从头开始选择
// 可以把[]hosts列表看成一个环形， 选择器在做圆周运动选择
func (s *RoundRobinSelector) Select() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.IsEOF() {
		s.lastIndex = 0
	} else {
		s.lastIndex++
	}
	return s.hosts[s.lastIndex]
}

// IsEOF 判断是否所有域名都选择过了一遍
// 如果是， 返回true, 否则返回false
func (s *RoundRobinSelector) IsEOF() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastIndex == len(s.hosts)-1 {
		return true
	}
	return false
}

// DomainGroup 是一组域名，分为主域名列表和备用域名列表
// 实现HostsSelector接口
type DomainGroup struct {
	// 主域名列表
	Main []string `json:"main,omitempty"`

	// 备用域名列表
	Backup []string `json:"backup:omitempty"`
}

// IsMainEmpty 判断是否Main域名列表为空
func (g *DomainGroup) IsMainEmpty() bool {
	return len(g.Main) <= 0
}

// IsBackupEmpty 判断backup域名列表是否为空
func (g *DomainGroup) IsBackupEmpty() bool {
	return len(g.Backup) <= 0
}

// IsEmpty 判断DomainGroup是否为空
// 如果g.Main和g.Backup域名列表都为空，那么返回true, 否则返回false
func (g *DomainGroup) IsEmpty() bool {
	if g.IsMainEmpty() && g.IsBackupEmpty() {
		return true
	}
	return false
}

// UpDomainGroup 表示一组域名， 分为src, acc, old_src, old_acc域名组
// src域名组为上传域名组， acc为加速上传域名组， old_src是老的上传域名组， old_acc是新的上传域名组
type UpDomainGroup struct {
	Src    DomainGroup `json:"src,omitempty"`
	Acc    DomainGroup `json:"acc,omitempty"`
	OldSrc DomainGroup `json:"old_src,omitempty"`
	OldAcc DomainGroup `json:"old_acc,omitempty"`
}

// IsEmpty 判断上传域名组信息是否为空
// 只有当Src, Acc, OldSrc, OldAcc都为空， 才返回true, 否则返回false
func (u *UpDomainGroup) IsEmpty() bool {
	if u.Src.IsEmpty() && u.Acc.IsEmpty() && u.OldSrc.IsEmpty() && u.OldAcc.IsEmpty() {
		return true
	}
	return false
}

// SelectUpDomainGroup 从新老上传域名组，加速上传域名组中选择一组返回。
// 如果新加速域名组Acc不为空， 就返回该域名组。
// 否则， 检查新的普通上传域名组， 如果不为空， 返回该域名组
// 最后以同样的逻辑检查老的普通域名组和老的加速上传域名组
func (u *UpDomainGroup) SelectUpDomainGroup() DomainGroup {
	if !u.Acc.IsEmpty() {
		return u.Acc
	}
	if !u.Src.IsEmpty() {
		return u.Src
	}
	if !u.OldAcc.IsEmpty() {
		return u.OldAcc
	}
	return u.OldSrc
}

// IoDomainGroup 表示存储下载入口域名组
type IoDomainGroup struct {
	// 最新的下载域名组
	Src DomainGroup `json:"src,omitempty"`

	// 老的下载域名组
	OldSrc DomainGroup `json:"old_src,omitempty"`
}

// IsEmpty 判断域名组信息是否为空
// 只有当Src, OldSrc都为空的时候才返回true, 否则返回false
func (i *IoDomainGroup) IsEmpty() bool {
	if i.Src.IsEmpty() && i.OldSrc.IsEmpty() {
		return true
	}
	return false
}

// RsDomainGroup 一般接口修改文件元信息用到
type RsDomainGroup struct {
	Src DomainGroup
}

// IsEmpty 判断Rs域名组信息是否为空
func (r *RsDomainGroup) IsEmpty() bool {
	if r.Src.IsEmpty() {
		return true
	}
	return false
}

// RsfDomainGroup 一般用于存储空间文件列举
type RsfDomainGroup struct {
	Src DomainGroup
}

// IsEmpty 判断Rsf域名组是否为空
func (rf *RsfDomainGroup) IsEmpty() bool {
	if rf.Src.IsEmpty() {
		return true
	}
	return false
}

// APIDomainGroup 一些接口有在使用该信息
type APIDomainGroup struct {
	Src DomainGroup
}

// IsEmpty 判断API域名组是否为空
func (a *APIDomainGroup) IsEmpty() bool {
	if a.Src.IsEmpty() {
		return true
	}
	return false
}

// RegionDomain 表示存储区域的域名组信息， 包括上传域名组，下载域名组等
// 存储区域分为华东，华南， 华北，东南亚， 北美， 每个存储区域都有相应的存储域名组信息
type RegionDomain struct {
	// 上传域名组
	Up UpDomainGroup `json:"up,omitempty"`

	// 下载域名组
	Io IoDomainGroup `json:"io,omitempty"`

	Rs RsDomainGroup `json:"rs,omitempty"`

	Rsf RsfDomainGroup `json:"rsf,omitempty"`

	API APIDomainGroup `json:"api,omitempty"`
}

// IsUpGroupEmpty 判断上传域名组是否为空
func (r *RegionDomain) IsUpGroupEmpty() bool {
	if r.Up.IsEmpty() {
		return true
	}
	return false
}

// IsIoGroupEmpty 判断下载域名组是否为空
func (r *RegionDomain) IsIoGroupEmpty() bool {
	if r.Io.IsEmpty() {
		return true
	}
	return false
}

// IsRsGroupEmpty 判断rs域名组是否为空
func (r *RegionDomain) IsRsGroupEmpty() bool {
	if r.Rs.IsEmpty() {
		return true
	}
	return false
}

// IsRsfGroupEmpty 判断rsf域名组是否为空
func (r *RegionDomain) IsRsfGroupEmpty() bool {
	if r.Rsf.IsEmpty() {
		return true
	}
	return false
}

// IsAPIGroupEmpty 判断API域名组是否为空
func (r *RegionDomain) IsAPIGroupEmpty() bool {
	if r.API.IsEmpty() {
		return true
	}
	return false
}

// RegionDomains 定义了和存储相关的域名信息, 包括上传域名组，下载域名组
type RegionDomains struct {
	Hosts []RegionDomain `json:"hosts,omitempty"`
}

// AllUpDomainGroupEmpty 判断是否所有的上传域名组都是空
func (r *RegionDomains) AllUpDomainGroupEmpty() bool {
	for _, rd := range r.Hosts {
		if !rd.Up.IsEmpty() {
			return false
		}
	}
	return true
}

// SelectUpDomainGroup 从域名组中选择一个上传域名组
func (r *RegionDomains) SelectUpDomainGroup() DomainGroup {
	for _, rd := range r.Hosts {
		if !rd.Up.IsEmpty() {
			if d := rd.Up.SelectUpDomainGroup(); !d.IsMainEmpty() {
				return d
			}
		}
	}
	return DomainGroup{}
}

// IsEmpty 判断域名组信息是否为空
func (r *RegionDomains) IsEmpty() bool {
	if len(r.Hosts) <= 0 {
		return true
	}
	return false
}

// Bucket 封装了存储区域和存储空间名字信息
type Bucket struct {
	Region string
	Name   string
}
