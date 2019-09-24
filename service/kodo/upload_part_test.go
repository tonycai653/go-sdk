package kodo_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/qiniu/go-sdk/qiniu/defs"
	"github.com/qiniu/go-sdk/service/kodo"
)

func TestMultipartUpload(t *testing.T) {
	kclient := newKodoClient(true, nil)
	testBucket := getTestBucket()
	data := strings.Repeat("testttttttttttttttttt", defs.MB)

	t.Run("multipart upload with data reader", func(t *testing.T) {
		input := &kodo.UploadInput{
			Data: strings.NewReader(data),
			Key:  "test.txt",
			// 覆盖上传
			PutPolicy: &kodo.PutPolicy{
				Scope: fmt.Sprintf("%s:%s", testBucket, "test.txt"),
			},
		}
		out := &kodo.UploadOutput{}
		if err := kclient.UploadMultipart(input, out); err != nil {
			if multiErr, ok := err.(kodo.MultiUploadFailure); ok {
				t.Fatalf("upload data error: %s, error code: %s, uploadID: %s", multiErr.Error(), multiErr.Code(), multiErr.UploadID())
			}
			t.Fatalf("upload data error: %#v\n", err)
		}
	})
}

func BenchmarkMultipartUpload(b *testing.B) {
	kclient := newKodoClient(true, nil)
	testBucket := getTestBucket()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		input := &kodo.UploadInput{
			Data: strings.NewReader(strings.Repeat("testtesttest", defs.MB)),
			Key:  "test.txt",
			// 覆盖上传
			PutPolicy: &kodo.PutPolicy{
				Scope: fmt.Sprintf("%s:%s", testBucket, "test.txt"),
			},
		}
		out := &kodo.UploadOutput{}
		b.StartTimer()
		if err := kclient.UploadMultipart(input, out); err != nil {
			if multiErr, ok := err.(kodo.MultiUploadFailure); ok {
				b.Fatalf("upload data error: %s, error code: %s, uploadID: %s", multiErr.Error(), multiErr.Code(), multiErr.UploadID())
			}
			b.Fatalf("upload data error: %#v\n", err)
		}
	}
}
