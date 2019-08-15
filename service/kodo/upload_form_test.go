package kodo_test

import (
	"strings"
	"testing"

	"github.com/qiniu/go-sdk/service/kodo"
)

func TestUploadForm(t *testing.T) {
	kclient := newKodoClient(true, true)
	testBucket := getTestBucket()

	t.Run("upload reader with putpolicy and region", func(t *testing.T) {
		out := &kodo.DefaultFormOutput{}
		input := &kodo.FormInput{
			Key:    "upload_form.txt",
			Region: "z0",
			Data:   strings.NewReader("hello world"),
			PutPolicy: &kodo.PutPolicy{
				Scope: testBucket + ":upload_form.txt",
			},
		}

		err := kclient.UploadForm(input, &out)
		if err != nil {
			t.Fatalf("Expected nil error, but got: %#v\n", err)
		}
		t.Log(out)
	})
	t.Run("upload file with putpolicy and region", func(t *testing.T) {
		out := &kodo.DefaultFormOutput{}
		input := &kodo.FormInput{
			Key:      "upload_form.txt",
			Filename: "testdata/upload_test.txt",
			Region:   "z0",
			PutPolicy: &kodo.PutPolicy{
				Scope: testBucket + ":upload_form.txt",
			},
		}
		err := kclient.UploadForm(input, &out)
		if err != nil {
			t.Fatalf("Expected nil error, but got: %#v\n", err)
		}
		t.Log(out)

	})
	t.Run("upload reader with upload token and region", func(t *testing.T) {
		policy := &kodo.PutPolicy{}
		policy.WithScope(testBucket, "upload_form.txt")
		upToken, err := policy.UploadToken(kclient.Config.Credentials)
		if err != nil {
			t.Fatalf("Expected nil error, but got: %#v\n", err)
		}

		out := &kodo.DefaultFormOutput{}
		input := &kodo.FormInput{
			Key:     "upload_form.txt",
			Region:  "z0",
			Data:    strings.NewReader("hello world"),
			UpToken: upToken,
		}

		err = kclient.UploadForm(input, &out)
		if err != nil {
			t.Fatalf("Expected nil error, but got: %#v\n", err)
		}
		t.Log(out)
	})
	t.Run("upload reader without upload token  and with region, bucket name set", func(t *testing.T) {
		out := &kodo.DefaultFormOutput{}
		input := &kodo.FormInput{
			BucketName: testBucket,
			Key:        "upload_form.txt",
			Region:     "z0",
			Data:       strings.NewReader("hello world"),
		}

		err := kclient.UploadForm(input, &out)
		if err != nil {
			t.Fatalf("Expected nil error, but got: %#v\n", err)
		}
		t.Log(out)
	})
	t.Run("upload reader with crc32", func(t *testing.T) {
		out := &kodo.DefaultFormOutput{}
		input := &kodo.FormInput{
			BucketName: testBucket,
			Key:        "upload_form.txt",
			Region:     "z0",
			Data:       strings.NewReader("hello world"),
			WithCrc32:  true,
		}

		err := kclient.UploadForm(input, &out)
		if err != nil {
			t.Fatalf("Expected nil error, but got: %#v\n", err)
		}
		t.Log(out)
	})
}
