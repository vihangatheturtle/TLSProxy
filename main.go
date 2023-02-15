package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	up "net/url"
	"strings"
)

func NewReq(method string, url string, payload []byte, chead map[string]string) ([]byte, error) {
	uparsed, err := up.Parse(url)
	if err != nil {
		return nil, err
	}

	headers := map[string][]string{
		"authority":          {uparsed.Host},
		"accept":             {"application/json, text/plain, */*"},
		"origin":             {"https://" + uparsed.Host + "/"},
		"referer":            {"https://" + uparsed.Host + "/"},
		"sec-ch-ua":          {"' Not A;Brand';v='99', 'Chromium';v='109', 'Google Chrome';v='109'"},
		"sec-ch-ua-mobile":   {"?0"},
		"sec-ch-ua-platform": {"'macOS'"},
		"sec-fetch-dest":     {"empty"},
		"sec-fetch-mode":     {"cors"},
		"sec-fetch-site":     {"same-site"},
		"user-agent":         {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36"},
	}

	for i, v := range chead {
		headers[i] = []string{v}
	}

	res, err := SendTLSRequest(
		strings.ToUpper(method),
		url,
		headers,
		nil,
	)

	return res, err
}

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(400)
		w.Write([]byte(`{"error": true, "message": "invalid_request_method"}`))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(`{"error": true, "message": "bad_request_body"}`))
		return
	}

	type request struct {
		Method  string `json:"method"`
		URL     string `json:"url"`
		Headers map[string]string
		Payload []byte `json:"payload"`
	}

	var data request

	err = json.Unmarshal(body, &data)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": true, "message": "request_parse_failed"}`))
		return
	}

	response, err := NewReq(data.Method, data.URL, data.Payload, data.Headers)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": true, "message": "proxied_request_failed"}`))
		return
	}

	w.WriteHeader(200)
	w.Write(response)
}

func main() {
	http.HandleFunc("/proxy", ProxyHandler)
	http.ListenAndServe("127.0.0.1:7738", nil)
}
