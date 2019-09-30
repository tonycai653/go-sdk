package kodo_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/qerr"
)

func TestKodoObjectManagement(t *testing.T) {
	kclient := newKodoClient(true, nil)

	t.Run("Kodo stat", func(t *testing.T) {
		_, err := kclient.Stat(getTestBucket(), "qiniu.png")

		if err != nil {
			t.Errorf("Expected nil stat error, but Got: %#v\n", err)
		}
	})

}

func TestRequestError(t *testing.T) {
	cfg := &qiniu.Config{
		RsHost: qiniu.String("http://localhost"),
	}
	kclient := newKodoClient(false, cfg)
	testBucket := getTestBucket()
	testKey := getTestKey()

	t.Run("Kodo invalid json response", func(t *testing.T) {
		statReq, _ := kclient.StatRequest(testBucket, testKey)
		ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			writer.WriteHeader(200)
			writer.Write([]byte("{\"Key\": \"test.txt\""))
		}))
		defer ts.Close()

		err := statReq.Send()
		if err == nil {
			t.Fatalf("Expected nonnil error\n")
		}
		if aerr, ok := err.(qerr.Error); !ok {
			t.Fatalf("Expected qerr.Error error, GOT %#v\n", err)
			if aerr.Code() != qerr.ErrCodeDeserialization {
				t.Fatalf("Expected %s error code, GOT %s\n", qerr.ErrCodeDeserialization, aerr.Code())
			}
		}
	})

	t.Run("Kodo unknown error", func(t *testing.T) {
		statReq, _ := kclient.StatRequest(testBucket, testKey)
		ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			writer.WriteHeader(200)
		}))
		defer ts.Close()

		err := statReq.Send()
		if aerr := err.(qerr.Error); aerr.Code() != qerr.ErrUnknown {
			t.Fatalf("Expected %s error code, GOT %s\n", qerr.ErrUnknown, aerr.Code())
		}
	})
}
