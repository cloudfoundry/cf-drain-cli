package cloudcontroller

import (
	"fmt"
	"io/ioutil"
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

type Curler interface {
	Curl(URL, method, body string) ([]byte, error)
}

type TokenFetcher interface {
	Token() (string, error)
}

func NewHTTPCurlClient(apiAddr string, d Doer, f TokenFetcher) *HTTPCurlClient {
	a, err := url.Parse(apiAddr)
	if err != nil {
		panic(err)
	}

	return &HTTPCurlClient{
		d: d,
		f: f,

		// save a copy so we can manipulate without races
		a: *a,
	}
}

func (c *HTTPCurlClient) Curl(url, method, body string) ([]byte, error) {
	if method == "GET" && body != "" {
		panic("GET method must not have a body")
	}

	token, err := c.f.Token()
	if err != nil {
		return nil, err
	}

	url = c.a.String() + url
	req, _ := http.NewRequest(method, url, ioutil.NopCloser(strings.NewReader(body)))
	req.Header.Set("Authorization", token)

	resp, err := c.d.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}
