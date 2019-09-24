package kodo_test

import (
	"os"

	"github.com/qiniu/go-sdk/qiniu"
	"github.com/qiniu/go-sdk/qiniu/session"
	"github.com/qiniu/go-sdk/service/kodo"
)

var (
	kodoClient *kodo.Kodo
)

const (
	defaultTestBucket = "gosdk"
	defaultTestKey    = "test.txt"
)

func newKodoClient(cache bool, cfg *qiniu.Config) *kodo.Kodo {
	if cache && kodoClient != nil {
		return kodoClient
	}
	s := session.Must(session.New(cfg))
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

func getTestKey() string {
	key := os.Getenv("QINIU_TEST_KEY")
	if key != "" {
		return key
	} else {
		return defaultTestKey
	}
}
