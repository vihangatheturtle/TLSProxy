package main

import (
	"bytes"
	"io"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/cookiejar"
	tls_client "github.com/bogdanfinn/tls-client"
)

func SendTLSRequest(method string, url string, headers map[string][]string, payload []byte, cjar ...*cookiejar.Jar) ([]byte, fhttp.Header, int, error) {
	var jar *cookiejar.Jar

	if len(cjar) == 0 {
		jar, _ = cookiejar.New(nil)
	} else {
		jar = cjar[0]
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(tls_client.Chrome_109),
		tls_client.WithCookieJar(jar),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)

	if err != nil {
		return nil, nil, 500, err
	}

	req, err := fhttp.NewRequest(method, url, bytes.NewBuffer(payload))

	if err != nil {
		return nil, nil, 500, err
	}

	req.Header = headers

	res, err := client.Do(req)

	if err != nil {
		return nil, nil, 500, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return nil, nil, 500, err
	}

	return body, res.Header, res.StatusCode, nil
}
