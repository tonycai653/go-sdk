package qiniu

import (
	"io"
	"sync"
)

// ReadSeekCloser 封装底层Reader, 使用实现Reader, Closer, Seeker接口
// 如果底层的数据不是Seeker, 那么Seek操作什么也不做，同时在请求遇到错误的时候也不会重试，直接返回失败
func ReadSeekCloser(r io.Reader) ReaderSeekerCloser {
	return ReaderSeekerCloser{r}
}

// ReaderSeekerCloser 实现了Reader, Seeker, Closer接口
type ReaderSeekerCloser struct {
	r io.Reader
}

// IsReaderSeekable 判断包裹的底层Reader是否是个Seeker
func IsReaderSeekable(r io.Reader) bool {
	switch v := r.(type) {
	case ReaderSeekerCloser:
		return v.IsSeeker()
	case *ReaderSeekerCloser:
		return v.IsSeeker()
	case io.ReadSeeker:
		return true
	default:
		return false
	}
}

// Read 读取数据到切片p
func (r ReaderSeekerCloser) Read(p []byte) (int, error) {
	switch t := r.r.(type) {
	case io.Reader:
		return t.Read(p)
	}
	return 0, nil
}

// Seek 设置并且返回下次读取的位置
//
// 当whence为0， offset是相对于数据开始的偏移量
// 当whence为1， offset是相对于当前读取位置的偏移量
// 当whence为2， offset是相对于数据末尾的偏移量
//
// 如果r的底层Reader不是可Seek的， 那么什么也不做，直接返回0， nil
func (r ReaderSeekerCloser) Seek(offset int64, whence int) (int64, error) {
	switch t := r.r.(type) {
	case io.Seeker:
		return t.Seek(offset, whence)
	}
	return int64(0), nil
}

// IsSeeker 判断是否底层的Reader是可Seek的
func (r ReaderSeekerCloser) IsSeeker() bool {
	_, ok := r.r.(io.Seeker)
	return ok
}

// HasLen 返回底层Reader的数据大小
// 如果底层Reader实现了Len方法， 调用该方法获取Reader可读的数据大小， 否则返回0
// 第二个参数bool表示是否获取成功, true - 成功， false - 失败
func (r ReaderSeekerCloser) HasLen() (int, bool) {
	type lenner interface {
		Len() int
	}

	if lr, ok := r.r.(lenner); ok {
		return lr.Len(), true
	}

	return 0, false
}

// GetLen 返回底层的Reader剩余未读数据大小
// 如果获取大小失败， 就返回-1
func (r ReaderSeekerCloser) GetLen() (int64, error) {
	if l, ok := r.HasLen(); ok {
		return int64(l), nil
	}

	if s, ok := r.r.(io.Seeker); ok {
		return seekerLen(s)
	}

	return -1, nil
}

// SeekerLen 判断Seeker剩余数据的小大， 返回相应的数据大小，或者错误
// 如果s是ReaderSeekerCloser， 并且封装的底层Reader是不可Seek的，那么返回-1, nil
func SeekerLen(s io.Seeker) (int64, error) {
	switch v := s.(type) {
	case ReaderSeekerCloser:
		return v.GetLen()
	case *ReaderSeekerCloser:
		return v.GetLen()
	}

	return seekerLen(s)
}

func seekerLen(s io.Seeker) (int64, error) {
	curOffset, err := s.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	endOffset, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	_, err = s.Seek(curOffset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	return endOffset - curOffset, nil
}

// Close 关闭ReaderSeekerCloser
func (r ReaderSeekerCloser) Close() error {
	switch t := r.r.(type) {
	case io.Closer:
		return t.Close()
	}
	return nil
}

// A WriteAtBuffer provides a in memory buffer supporting the io.WriterAt interface
// Can be used with the s3manager.Downloader to download content to a buffer
// in memory. Safe to use concurrently.
type WriteAtBuffer struct {
	buf []byte
	m   sync.Mutex

	// GrowthCoeff defines the growth rate of the internal buffer. By
	// default, the growth rate is 1, where expanding the internal
	// buffer will allocate only enough capacity to fit the new expected
	// length.
	GrowthCoeff float64
}

// NewWriteAtBuffer creates a WriteAtBuffer with an internal buffer
// provided by buf.
func NewWriteAtBuffer(buf []byte) *WriteAtBuffer {
	return &WriteAtBuffer{buf: buf}
}

// WriteAt writes a slice of bytes to a buffer starting at the position provided
// The number of bytes written will be returned, or error. Can overwrite previous
// written slices if the write ats overlap.
func (b *WriteAtBuffer) WriteAt(p []byte, pos int64) (n int, err error) {
	pLen := len(p)
	expLen := pos + int64(pLen)
	b.m.Lock()
	defer b.m.Unlock()
	if int64(len(b.buf)) < expLen {
		if int64(cap(b.buf)) < expLen {
			if b.GrowthCoeff < 1 {
				b.GrowthCoeff = 1
			}
			newBuf := make([]byte, expLen, int64(b.GrowthCoeff*float64(expLen)))
			copy(newBuf, b.buf)
			b.buf = newBuf
		}
		b.buf = b.buf[:expLen]
	}
	copy(b.buf[pos:], p)
	return pLen, nil
}

// Bytes returns a slice of bytes written to the buffer.
func (b *WriteAtBuffer) Bytes() []byte {
	b.m.Lock()
	defer b.m.Unlock()
	return b.buf
}
