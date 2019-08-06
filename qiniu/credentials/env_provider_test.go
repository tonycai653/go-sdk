package credentials

import (
	"os"
	"testing"
)

func TestEnvProviderRetrieve(t *testing.T) {
	os.Clearenv()
	os.Setenv("QINIU_ACCESS_KEY_ID", "access")
	os.Setenv("QINIU_SECRET_ACCESS_KEY", "secret")

	e := EnvProvider{}
	creds, err := e.Retrieve()
	if err != nil {
		t.Errorf("EnvProvider Retrieve should error should be nil, Got: %v\n", err)
	}
	if creds.AccessKey != "access" {
		t.Errorf("Expect access key to match, Expected: %s, Got: %s\n", "access", creds.AccessKey)
	}
	if string(creds.SecretKey) != "secret" {
		t.Errorf("Expect secret key to match, Expected: %s, Got: %s\n", "secret", string(creds.SecretKey))
	}
}

func TestEnvProviderNoAccessKeyID(t *testing.T) {
	os.Clearenv()
	os.Setenv("QINIU_SECRET_ACCESS_KEY", "secret")

	e := EnvProvider{}
	_, err := e.Retrieve()

	if err != ErrAccessKeyIDNotFound {
		t.Errorf("ErrAccessKeyIDNotFound expected, but was %#v\n", err)
	}
}

func TestEnvProviderNoSecretAccessKey(t *testing.T) {
	os.Clearenv()
	os.Setenv("QINIU_ACCESS_KEY_ID", "access")

	e := EnvProvider{}
	_, err := e.Retrieve()

	if err != ErrSecretAccessKeyNotFound {
		t.Errorf("ErrSecretAccessKeyNotFound expected, but was %#v\n", err)
	}
}

func TestEnvProviderAlternateNames(t *testing.T) {
	os.Clearenv()
	os.Setenv("QINIU_ACCESS_KEY", "access")
	os.Setenv("QINIU_SECRET_KEY", "secret")

	e := EnvProvider{}
	creds, err := e.Retrieve()
	if err != nil {
		t.Errorf("Expected nil error, Got: %#v\n", err)
	}
	if creds.AccessKey != "access" {
		t.Errorf("Expect access key to match, Expected: %s, Got: %s\n", "access", creds.AccessKey)
	}
	if string(creds.SecretKey) != "secret" {
		t.Errorf("Expect secret key to match, Expected: %s, Got: %s\n", "secret", string(creds.SecretKey))
	}
}
