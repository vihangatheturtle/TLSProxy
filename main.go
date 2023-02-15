package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	up "net/url"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/cookiejar"
)

var cookies *cookiejar.Jar

func NewReq(method string, url string, payload string, chead map[string]string, dontIncludeOptionalHeaders bool) ([]byte, fhttp.Header, int, error) {
	uparsed, err := up.Parse(url)
	if err != nil {
		return nil, nil, 500, err
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
		"user-agent":         {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.4896.127 Safari/537.36"},
	}

	if dontIncludeOptionalHeaders {
		headers = map[string][]string{
			"user-agent": {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.4896.127 Safari/537.36"},
		}
	}

	for i, v := range chead {
		headers[i] = []string{v}
	}

	var rpayload []byte = nil

	if payload != "" {
		rpayload = []byte(payload)
	}

	// log.Println(strings.ToUpper(method),
	// 	url,
	// 	headers,
	// 	string(rpayload))

	res, headers, status, err := SendTLSRequest(
		strings.ToUpper(method),
		url,
		headers,
		rpayload,
		cookies,
	)

	return res, headers, status, err
}

func GetCookie(w http.ResponseWriter, r *http.Request) {
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
		URL string `json:"url"`
	}

	var data request

	err = json.Unmarshal(body, &data)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": true, "message": "request_parse_failed"}`))
		return
	}

	if data.URL == "" {
		w.WriteHeader(400)
		w.Write([]byte(`{"error": true, "message": "no_url_provided"}`))
		return
	}

	parsedURL, err := up.Parse(data.URL)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(`{"error": true, "message": "bad_url"}`))
		return
	}

	if cookies != nil {
		cs := cookies.Cookies(parsedURL)

		if cs == nil {
			w.WriteHeader(200)
			w.Write([]byte(`[]`))
			return
		}

		data, err := json.Marshal(cs)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error": true, "message": "cookies_parse_failed"}`))
			return
		}

		w.WriteHeader(200)
		w.Write(data)
		return
	}
}

func ResetCookies(w http.ResponseWriter, r *http.Request) {
	cj, err := cookiejar.New(nil)
	if err == nil {
		cookies = cj
		w.WriteHeader(200)
		w.Write([]byte(`{"error": false, "message": "cookies_reset"}`))
	} else {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": true, "message": "cookie_reset_failed"}`))
	}
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
		Method         string `json:"method"`
		URL            string `json:"url"`
		Headers        map[string]string
		Payload        string `json:"payload"`
		Body           string `json:"body"`
		UseBaseHeaders bool   `json:"useBaseHeaders"`
	}

	var data request

	err = json.Unmarshal(body, &data)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": true, "message": "request_parse_failed"}`))
		return
	}

	if data.URL == "" {
		w.WriteHeader(400)
		w.Write([]byte(`{"error": true, "message": "no_url_provided"}`))
		return
	}

	if data.Method == "" {
		data.Method = "GET"
	}

	if data.Payload == "" {
		data.Payload = data.Body
	}

	if data.Payload != "" && data.Method == "GET" {
		data.Method = "POST"
	}

	response, headers, status, err := NewReq(data.Method, data.URL, data.Payload, data.Headers, data.UseBaseHeaders)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": true, "message": "proxied_request_failed"}`))
		return
	}

	if headers != nil {
		shdrs, _ := headers.SortedKeyValues(nil)
		for i := 0; i < len(shdrs); i++ {
			if shdrs[i].Key == "Content-Encoding" || shdrs[i].Key == "Content-Length" {
				continue
			}
			w.Header().Set(shdrs[i].Key, shdrs[i].Values[0])
		}
	}

	w.WriteHeader(status)
	w.Write(response)
}

func main() {
	cj, err := cookiejar.New(nil)
	if err == nil {
		cookies = cj
	}
	http.HandleFunc("/proxy", ProxyHandler)
	http.HandleFunc("/reset-cookies", ResetCookies)
	http.HandleFunc("/get-cookies", GetCookie)
	http.ListenAndServe("127.0.0.1:7738", nil)
}
