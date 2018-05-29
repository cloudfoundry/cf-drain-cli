package cloudcontroller

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type HTTPCurlClient struct {
	d Doer
	f TokenFetcher
	a url.URL
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type TokenFetcher interface {
	Token() (string, string, error)
}

func NewHTTPCurlClient(apiAddr string, d Doer, f TokenFetcher) *HTTPCurlClient {
	a, err := url.Parse(apiAddr)
	if err != nil {
		log.Fatalf("failed to parse CAPI address: %s", err)
	}

	return &HTTPCurlClient{
		d: d,
		f: f,

		// save a copy so we can manipulate without races
		a: *a,
	}
}

func NewHTTPAuthCurlClient(apiAddr string, d Doer) *HTTPCurlClient {
	a, err := url.Parse(apiAddr)
	if err != nil {
		log.Fatalf("failed to parse CAPI address: %s", err)
	}

	return &HTTPCurlClient{
		d: d,

		// save a copy so we can manipulate without races
		a: *a,
	}
}

func (c *HTTPCurlClient) Curl(url, method, body string) ([]byte, error) {
	accessToken, _, err := c.f.Token()
	if err != nil {
		return nil, err
	}

	return c.AuthCurl(url, method, body, accessToken)
}

func (c *HTTPCurlClient) AuthCurl(url, method, body, token string) ([]byte, error) {
	if method == http.MethodGet && body != "" {
		log.Panic("GET method must not have a body")
	}

	url = c.a.String() + url
	req, _ := http.NewRequest(method, url, ioutil.NopCloser(strings.NewReader(body)))
	req.Header.Set("Authorization", token)

	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.d.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, data)
	}

	return data, nil
}
