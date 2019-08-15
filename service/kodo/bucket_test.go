package kodo_test

import (
	"testing"
)

func TestKodoObjectManagement(t *testing.T) {
	kclient := newKodoClient(true, false)

	_, err := kclient.Stat(getTestBucket(), "qiniu.png")

	if err != nil {
		t.Errorf("Expected nil stat error, but Got: %#v\n", err)
	}

}
