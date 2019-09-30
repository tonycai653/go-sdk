package qiniu_test

import (
	"io"
	"strings"
	"testing"

	"github.com/qiniu/go-sdk/qiniu"
)

type unseekableReader struct {
	r io.Reader
}

func (u *unseekableReader) Read(p []byte) (int, error) {
	return u.r.Read(p)
}

func newUnseekableReader(s string) *unseekableReader {
	return &unseekableReader{
		r: strings.NewReader(s),
	}
}

func TestSeekerLen(t *testing.T) {
	r := qiniu.ReadSeekCloser(newUnseekableReader("hello world"))
	if yes := r.IsSeeker(); yes {
		t.Fatalf("Expected nonseekable reader\n")
	}
	if n, err := qiniu.SeekerLen(r); n != -1 || err != nil {
		t.Fatalf("Expected -1 and nil error for nonseekable reader, GOT size: %d, err: %#v\n", n, err)
	}
}
