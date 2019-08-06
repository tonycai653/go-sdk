package kodo_test

import (
	"os"

	"github.com/qiniu/go-sdk/qiniu/session"
	"github.com/qiniu/go-sdk/service/kodo"
)

var (
	kodoClient *kodo.Kodo
)

const (
	defaultTestBucket = "gosdk"
)

func newKodoClient(cache bool) *kodo.Kodo {
	if cache && kodoClient != nil {
		return kodoClient
	}
	s := session.Must(session.New())

	kodoClient = kodo.New(s)
	return kodoClient
}

func getTestBucket() string {
	testBucket := os.Getenv("QINIU_TEST_BUCKET")
	if testBucket != "" {
		return testBucket
	} else {
		return defaultTestBucket
	}
}
