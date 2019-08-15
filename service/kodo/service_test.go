package kodo_test

import (
	"testing"
)

func TestQueryBucketIoUpDomains(t *testing.T) {
	kclient := newKodoClient(true, true)

	ioUpDomains, err := kclient.QueryRegionDomains(getTestBucket())
	if err != nil {
		t.Fatalf("Expected nil error, but got: %#v\n", err)
	}
	t.Log(ioUpDomains)
}
