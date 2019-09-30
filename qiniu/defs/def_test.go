package defs

import (
	"testing"
)

func TestSize(t *testing.T) {
	cases := []struct {
		Size   Size
		String string
	}{
		{Size: 100, String: "100B"},
		{Size: 1024, String: "1.00KB"},
		{Size: 102400, String: "100.00KB"},
		{Size: 2 * 1024 * 1024, String: "2.00MB"},
		{Size: 2 * 1024 * 1024 * 1024, String: "2.00GB"},
		{Size: 2 * 1024 * 1024 * 1024 * 1024, String: "2.00PB"},
		{Size: 24641536, String: "23.50MB"},
	}
	for _, c := range cases {
		if c.Size.String() != c.String {
			t.Fatalf("Expected size representation: `%s`, GOT: `%s`\n", c.String, c.Size.String())
		}
	}
}
