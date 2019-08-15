package credentials

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBytesFromRequest(t *testing.T) {
	// 只有content-length和body大小对测试有影响
	// http method, URL不用设置
	reqPrams := []ReqParams{
		{
			Method:  "",
			URL:     "",
			Headers: http.Header{"Content-Length": []string{"0"}},
			Body:    strings.NewReader(""),
		},
		{
			Method:  "",
			URL:     "",
			Headers: http.Header{"Content-Length": []string{"-1"}},
			Body:    strings.NewReader("h"),
		},
		{
			Method:  "",
			URL:     "",
			Headers: http.Header{"Content-Length": []string{"5"}},
			Body:    strings.NewReader("testt"),
		},
	}
	reqs, err := genRequests(reqPrams)
	if err != nil {
		t.Fatal("Failed to generate requests")
	}
	testCases := []struct {
		Req      *http.Request
		Expected string
	}{
		{
			Req:      reqs[0],
			Expected: "",
		},
		{
			Req:      reqs[1],
			Expected: "h",
		},
		{
			Req:      reqs[2],
			Expected: "testt",
		},
	}
	for _, v := range testCases {
		bs, err := bytesFromRequest(v.Req)
		if err != nil {
			t.Error("Get bytes from request error", err)
		}
		if v.Expected != string(bs) {
			t.Errorf("Expected: %s, Got: %s", v.Expected, string(bs))
		}
	}
}

func TestValueEmpty(t *testing.T) {
	testCases := []struct {
		Value    Value
		Expected bool
	}{
		{
			Value: Value{
				AccessKey: "",
				SecretKey: []byte(""),
			},
			Expected: true,
		},
		{
			Value: Value{
				AccessKey: "",
				SecretKey: []byte("test"),
			},
			Expected: true,
		},
		{
			Value: Value{
				AccessKey: "test",
				SecretKey: []byte(""),
			},
			Expected: true,
		},
		{
			Value: Value{
				AccessKey: "tst",
				SecretKey: []byte("test1"),
			},
			Expected: false,
		},
	}

	for _, v := range testCases {
		if v.Value.IsEmpty() != v.Expected {
			t.Errorf("Expected = %v, Got = %v\n", v.Expected, v.Value.IsEmpty())
		}
	}
}

func TestCredentials(t *testing.T) {
	// 准备工作，获取密钥， 后面的子测试要用到
	cred := New("ak", "sk")
	value, err := cred.Get()
	if err != nil {
		t.Fatal("failed to get credentials value")
	}

	t.Run("credentials new", func(t *testing.T) {
		if value.AccessKey != "ak" || string(value.SecretKey) != "sk" {
			t.Fatalf("Expected accessKey, secretKey: %s, %s, Got: %s, %s\n", "ak", "sk", value.AccessKey, string(value.SecretKey))
		}
	})
	t.Run("credentials sign", func(t *testing.T) {
		testStrs := []struct {
			Data   string
			Signed string
		}{
			{Data: "hello", Signed: "ak:NDN8cM0rwosxhHJ6QAcI7ialr0g="},
			{Data: "world", Signed: "ak:wZ-sw41ayh3PFDmQA-D3o7eBJIY="},
			{Data: "-test", Signed: "ak:oJ59sZasiWSqL1o7ugZs5OInEK4="},
			{Data: "ba#a-", Signed: "ak:tqHL8V2BbNI0dVDXsvueZp_2QnI="},
		}

		for _, b := range testStrs {
			got := value.Sign([]byte(b.Data))
			if got != b.Signed {
				t.Errorf("Sign %q, Expected=%q, Got=%q\n", b.Data, b.Signed, got)
			}
		}
	})
	t.Run("credentials sign with data", func(t *testing.T) {
		testStrs := []struct {
			Data   string
			Signed string
		}{
			{Data: "hello", Signed: "ak:2pn0qs-2kfEsQFuHI2pAYlo0hpc=:aGVsbG8="},
			{Data: "world", Signed: "ak:vzqcP6VeODVu_youBJnyr_nefT4=:d29ybGQ="},
			{Data: "-test", Signed: "ak:uV60zWZgj-Jbrg9VHc06Nok64Bw=:LXRlc3Q="},
			{Data: "ba#a-", Signed: "ak:RLvTUx_kizrrbpSrinkdxC4jCy8=:YmEjYS0="},
		}
		for _, b := range testStrs {
			got := value.SignWithData([]byte(b.Data))
			if got != b.Signed {
				t.Errorf("SignWithData %q, Expected=%q, Got=%q\n", b.Data, b.Signed, got)
			}
		}
	})
	t.Run("credentials sign request", func(t *testing.T) {
		inputs := []ReqParams{
			{Method: "", URL: "", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "", URL: "", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "GET", URL: "", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "POST", URL: "", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "", URL: "http://upload.qiniup.com", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "", URL: "http://upload.qiniup.com", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "", URL: "http://upload.qiniup.com", Headers: http.Header{"Content-Type": []string{"application/x-www-form-URLencoded"}}, Body: strings.NewReader(`name=test&language=go`)},
		}
		wants := []string{
			"ak:qfWnqF1E_vfzjZnReCVkcSMl29M=",
			"ak:qfWnqF1E_vfzjZnReCVkcSMl29M=",
			"ak:qfWnqF1E_vfzjZnReCVkcSMl29M=",
			"ak:qfWnqF1E_vfzjZnReCVkcSMl29M=",
			"ak:qfWnqF1E_vfzjZnReCVkcSMl29M=",
			"ak:qfWnqF1E_vfzjZnReCVkcSMl29M=",
			"ak:h8gBb1Adb2Jgoys1N8sRVAnNvpw=",
		}
		reqs, gErr := genRequests(inputs)
		if gErr != nil {
			t.Errorf("generate requests: %v\n", gErr)
		}
		for ind, req := range reqs {
			got, sErr := value.SignRequest(req)
			if sErr != nil {
				t.Errorf("SignRequest: %v\n", sErr)
			}
			if got != wants[ind] {
				t.Errorf("SignRequest, Expected = %q, Got = %q\n", wants[ind], got)
			}
		}
	})

	t.Run("credentials sign request v2", func(t *testing.T) {
		inputs := []ReqParams{
			{Method: "", URL: "", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "", URL: "", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "GET", URL: "", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "POST", URL: "", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "", URL: "http://upload.qiniup.com", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "", URL: "http://upload.qiniup.com", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
			{Method: "", URL: "http://upload.qiniup.com", Headers: http.Header{"Content-Type": []string{"application/x-www-form-URLencoded"}}, Body: strings.NewReader(`name=test&language=go`)},
		}
		wants := []string{
			"ak:XNay-AIghhXfytRKsKNj0DQqV2E=",
			"ak:K1DI0goT05yhGizDFE5FiPJxAj4=",
			"ak:XNay-AIghhXfytRKsKNj0DQqV2E=",
			"ak:0ujEjW_vLRZxebsveBgqa3JyQ-w=",
			"ak:Eadl-_gKUNECGo3mcikiTBoNfqI=",
			"ak:Pkuq20x3HNWJlHDbRLW1kDYmXF0=",
			"ak:rZjOJKtlePVSegqoSO8p6Gpsr64=",
		}
		reqs, gErr := genRequests(inputs)
		if gErr != nil {
			t.Errorf("generate requests: %v\n", gErr)
		}
		for ind, req := range reqs {
			got, sErr := value.SignRequestV2(req)
			if sErr != nil {
				t.Errorf("SignRequest: %v\n", sErr)
			}
			if got != wants[ind] {
				t.Errorf("SignRequest, want = %q, got = %q\n", wants[ind], got)
			}
		}
	})
}

func TestCollectData(t *testing.T) {
	inputs := []ReqParams{
		{Method: "", URL: "", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "", URL: "", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "GET", URL: "", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "POST", URL: "", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "", URL: "http://upload.qiniup.com", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "", URL: "http://upload.qiniup.com", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "", URL: "http://upload.qiniup.com", Headers: http.Header{"Content-Type": []string{"application/x-www-form-URLencoded"}}, Body: strings.NewReader(`name=test&language=go`)},
		{Method: "", URL: "http://upload.qiniup.com?v=2", Headers: http.Header{"Content-Type": []string{"application/x-www-form-URLencoded"}}, Body: strings.NewReader(`name=test&language=go`)},
		{Method: "", URL: "http://upload.qiniup.com/find/sdk?v=2", Headers: http.Header{"Content-Type": []string{"application/x-www-form-URLencoded"}}, Body: strings.NewReader(`name=test&language=go`)},
	}
	wants := []string{"\n", "\n", "\n", "\n", "\n", "\n", "\nname=test&language=go", "?v=2\nname=test&language=go", "/find/sdk?v=2\nname=test&language=go"}
	reqs, gErr := genRequests(inputs)
	if gErr != nil {
		t.Errorf("generate requests: %v\n", gErr)
	}

	for ind, req := range reqs {
		data, err := collectData(req)
		if err != nil {
			t.Error("collectData: ", err)
		}
		if string(data) != wants[ind] {
			t.Errorf("collectData, Expected = %q, Got = %q\n", wants[ind], data)
		}
	}

}

func TestCollectDataV2(t *testing.T) {
	inputs := []ReqParams{
		{Method: "", URL: "", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "", URL: "", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "", URL: "", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "GET", URL: "", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "POST", URL: "", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "", URL: "http://upload.qiniup.com", Headers: nil, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "", URL: "http://upload.qiniup.com", Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: strings.NewReader(`{"name": "test"}`)},
		{Method: "", URL: "http://upload.qiniup.com", Headers: http.Header{"Content-Type": []string{"application/x-www-form-URLencoded"}}, Body: strings.NewReader(`name=test&language=go`)},
		{Method: "", URL: "http://upload.qiniup.com?v=2", Headers: http.Header{"Content-Type": []string{"application/x-www-form-URLencoded"}}, Body: strings.NewReader(`name=test&language=go`)},
		{Method: "", URL: "http://upload.qiniup.com/find/sdk?v=2", Headers: http.Header{"Content-Type": []string{"application/x-www-form-URLencoded"}}, Body: strings.NewReader(`name=test&language=go`)},
	}

	wants := []string{
		"GET \nHost: \nContent-Type: application/json\n\n{\"name\": \"test\"}",
		"GET \nHost: \n\n",
		"GET \nHost: \nContent-Type: application/json\n\n{\"name\": \"test\"}",
		"GET \nHost: \n\n",
		"POST \nHost: \nContent-Type: application/json\n\n{\"name\": \"test\"}",
		"GET \nHost: upload.qiniup.com\n\n",
		"GET \nHost: upload.qiniup.com\nContent-Type: application/json\n\n{\"name\": \"test\"}",
		"GET \nHost: upload.qiniup.com\nContent-Type: application/x-www-form-URLencoded\n\nname=test&language=go",
		"GET ?v=2\nHost: upload.qiniup.com\nContent-Type: application/x-www-form-URLencoded\n\nname=test&language=go",
		"GET /find/sdk?v=2\nHost: upload.qiniup.com\nContent-Type: application/x-www-form-URLencoded\n\nname=test&language=go",
	}
	reqs, gErr := genRequests(inputs)
	if gErr != nil {
		t.Errorf("generate requests: %v\n", gErr)
	}

	for ind, req := range reqs {
		data, err := collectDataV2(req)
		if err != nil {
			t.Error("collectDataV2: ", err)
		}
		if string(data) != wants[ind] {
			t.Errorf("collectDataV2, Expected = %q, Got = %q\n", wants[ind], data)
		}
	}
}

func genRequest(method, URL string, headers http.Header, body io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequest(method, URL, body)
	if err != nil {
		return
	}
	req.Header = headers
	return
}

type ReqParams struct {
	Method  string
	URL     string
	Headers http.Header
	Body    io.Reader
}

func genRequests(params []ReqParams) (reqs []*http.Request, err error) {
	for _, reqParam := range params {
		req, rErr := genRequest(reqParam.Method, reqParam.URL, reqParam.Headers, reqParam.Body)
		if rErr != nil {
			err = rErr
			return
		}
		reqs = append(reqs, req)
	}
	return
}
