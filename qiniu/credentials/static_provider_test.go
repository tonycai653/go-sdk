package credentials

import (
	"testing"
)

func TestStaticProviderGet(t *testing.T) {
	s := StaticProvider{
		Value: Value{
			AccessKey: "AKID",
			SecretKey: []byte("SECRET"),
		},
	}

	creds, err := s.Retrieve()

	if err != nil {
		t.Errorf("Expected nil error, but Got: %#v\n", err)
	}

	if creds.AccessKey != "AKID" {
		t.Errorf("Expect AccessKey to match, Expected: %s, Got: %s\n", "AKID", creds.AccessKey)
	}
	if string(creds.SecretKey) != "SECRET" {
		t.Errorf("Expect SecretKey to match, Expected: %s, but Got: %s\n", "SECRET", string(creds.SecretKey))
	}
}

func TestStaticProviderEmpty(t *testing.T) {
	emptyProviders := []StaticProvider{
		{
			Value: Value{
				AccessKey: "",
				SecretKey: []byte("he"),
			},
		},
		{
			Value: Value{
				AccessKey: "tet",
				SecretKey: []byte(""),
			},
		},
		{
			Value: Value{
				AccessKey: "",
				SecretKey: []byte(""),
			},
		},
	}

	for _, s := range emptyProviders {
		_, err := s.Retrieve()
		if err != ErrStaticCredentialsEmpty {
			t.Errorf("Expected ErrStaticCredentialsEmpty, but Got: %#v\n", err)
		}
	}
}

func TestStaticProviderFromCreds(t *testing.T) {
	creds := NewStaticCredentialsFromCreds(Value{
		AccessKey: "ak",
		SecretKey: []byte("sk"),
	})
	if creds == nil {
		t.Errorf("Expected none nil Credentials pointer, but Got nil\n")
	}
}
