/*
 *    Copyright [2020] Sergey Kudasov
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package loadgen

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"strings"
	"time"
)

const (
	RequestHeader      = "========== REQUEST ==========\n%s\n"
	RequestHeaderBody  = "========== REQUEST ==========\n%s\n%s\n"
	ResponseHeaderBody = "========== RESPONSE ==========\n%s\n%s\n"
	ResponseHeader     = "========== RESPONSE ==========\n%s\n"
	HTTPBodyDelimiter  = "\r\n\r\n"
)

// DumpTransport log http request/responses, pprint bodies
type DumpTransport struct {
	r http.RoundTripper
}

func (d *DumpTransport) RoundTrip(h *http.Request) (*http.Response, error) {
	var respString string
	var pprintBody string
	h.Header.Set("X-Content-Type-Options", "nosniff")
	dump, _ := httputil.DumpRequestOut(h, true)
	if bodyIsJson(h.Header) {
		req, pprintBody := d.prettyPrintJsonBody(dump)
		fmt.Printf(RequestHeaderBody, req, pprintBody)
	} else {
		fmt.Printf(RequestHeader, dump)
	}
	resp, err := d.r.RoundTrip(h)
	if err != nil {
		return nil, err
	}
	if resp != nil && resp.Body != nil && bodyIsJson(resp.Header) {
		defer resp.Body.Close()
		dump, _ = httputil.DumpResponse(resp, true)
		respString, pprintBody = d.prettyPrintJsonBody(dump)
		fmt.Printf(ResponseHeaderBody, respString, pprintBody)
		return resp, err
	}
	dump, _ = httputil.DumpResponse(resp, true)
	fmt.Printf(ResponseHeader, dump)
	return resp, err
}

// prettyPrintJsonBody returns http format request and pretty printed json body
func (d *DumpTransport) prettyPrintJsonBody(b []byte) (string, string) {
	var pprintBody []byte
	s := string(b)
	sp := strings.Split(s, HTTPBodyDelimiter)
	if len(sp) == 2 {
		body := sp[1]
		if strings.HasPrefix(body, "[") {
			var objmap []*json.RawMessage
			err := json.Unmarshal([]byte(body), &objmap)
			if err != nil {
				log.Fatal(err)
			}

			pprintBody, err = json.MarshalIndent(objmap, "", "    ")
			if err != nil {
				log.Fatal(err)
			}
			return sp[0], string(pprintBody)
		}
		var objmap map[string]*json.RawMessage
		err := json.Unmarshal([]byte(body), &objmap)
		if err != nil {
			log.Fatal(err)
		}
		pprintBody, err = json.MarshalIndent(objmap, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
	}
	return sp[0], string(pprintBody)
}

// NewLoggintHTTPClient creates new client with debug http
func NewLoggingHTTPClient(debug bool, transportTimeout int) *http.Client {
	var transport http.RoundTripper
	if debug {
		transport = &DumpTransport{
			http.DefaultTransport,
		}
	} else {
		transport = http.DefaultTransport
	}
	cookieJar, _ := cookiejar.New(nil)
	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(transportTimeout) * time.Second,
		Jar:       cookieJar,
	}
}

func bodyIsJson(h http.Header) bool {
	return strings.Contains(h.Get("content-type"), "application/json")
}
