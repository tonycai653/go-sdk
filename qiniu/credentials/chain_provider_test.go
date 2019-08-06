package credentials

import (
	"reflect"
	"testing"

	"github.com/qiniu/go-sdk/qiniu/qerr"
)

type stubProvider struct {
	creds Value
	err   error
}

func (s *stubProvider) Retrieve() (Value, error) {
	s.creds.ProviderName = "stubProvider"
	return s.creds, s.err
}

func TestChainProviderWithNames(t *testing.T) {
	p := &ChainProvider{
		Providers: []Provider{
			&stubProvider{err: qerr.New("FirstError", "first provider error", nil)},
			&stubProvider{err: qerr.New("SecondError", "second provider error", nil)},
			&stubProvider{
				creds: Value{
					AccessKey: "AKIF",
					SecretKey: []byte("NOSECRET"),
				},
			},
			&stubProvider{
				creds: Value{
					AccessKey: "AKID",
					SecretKey: []byte("SECRET"),
				},
			},
		},
	}

	creds, err := p.Retrieve()
	if err != nil {
		t.Errorf("Expect no err, Got error: %v\n", err)
	}
	if creds.ProviderName != "stubProvider" {
		t.Errorf("creds.ProviderName not match, Expected: %s, Got: %s", "stubProvider", creds.ProviderName)
	}
	if creds.AccessKey != "AKIF" {
		t.Errorf("AccessKey not match, Expected: %s, Got: %s", "AKIF", creds.AccessKey)
	}
	if string(creds.SecretKey) != "NOSECRET" {
		t.Errorf("SecretKey not match, Expected: %s, Got: %s", "NOSECRET", string(creds.SecretKey))
	}
}

func TestChainProviderWithNoValidProvider(t *testing.T) {
	p := &ChainProvider{
		Providers: []Provider{
			&stubProvider{err: qerr.New("FirstError", "first provider error", nil)},
			&stubProvider{err: qerr.New("SecondError", "second provider error", nil)},
		},
	}

	_, err := p.Retrieve()
	if err != ErrNoValidProvidersFoundInChain {
		t.Errorf("Expect no valid proviers error returned, Got: %v\n", err)
	}
}

func TestChainProviderWithNoProvider(t *testing.T) {
	p := &ChainProvider{
		Providers: []Provider{},
	}

	_, err := p.Retrieve()
	if err != ErrNoValidProvidersFoundInChain {
		t.Errorf("Expect no valid proviers error returned, Got: %v\n", err)
	}
}

func TestChainProviderWithNoValidProviderWithVerboseEnabled(t *testing.T) {
	errs := []error{
		qerr.New("FirstError", "first provider error", nil),
		qerr.New("SecondError", "second provider error", nil),
	}
	p := &ChainProvider{
		VerboseErrors: true,
		Providers: []Provider{
			&stubProvider{err: errs[0]},
			&stubProvider{err: errs[1]},
		},
	}

	_, err := p.Retrieve()
	if !reflect.DeepEqual(err, qerr.NewBatchError("NoCredentialProviders", "no valid providers in chain", errs)) {
		t.Errorf("Expect NoCredentialProviders error returned, Got: %v\n", err)
	}
}
