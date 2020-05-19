package cloudcontroller

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type HTTPCurlClient struct {
	d Doer
	f TokenFetcher
	r SaveAndRestager
	a string

	mu          sync.RWMutex
	accessToken string
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type TokenFetcher interface {
	Token() (string, string, error)
}

type SaveAndRestager interface {
	SaveAndRestage(refreshToken string)
}

type SaveAndRestagerFunc func(string)

func (f SaveAndRestagerFunc) SaveAndRestage(refToken string) {
	f(refToken)
}

func NewHTTPCurlClient(apiAddr string, d Doer, f TokenFetcher, r SaveAndRestager) *HTTPCurlClient {
	return &HTTPCurlClient{d: d, f: f, a: apiAddr, r: r}
}

func (c *HTTPCurlClient) Curl(url, method, body string) ([]byte, error) {
	accToken, _, err := c.token()
	if err != nil {
		return nil, err
	}

	return c.authCurl(url, method, body, accToken)
}

func (c *HTTPCurlClient) authCurl(URL, method, body, token string) ([]byte, error) {
	if method == http.MethodGet && body != "" {
		log.Panic("GET method must not have a body")
	}

	u, err := url.Parse(c.a)
	if err != nil {
		log.Fatalf("failed to parse CAPI address: %s", err)
	}
	URL = u.String() + URL

	req, _ := http.NewRequest(method, URL, ioutil.NopCloser(strings.NewReader(body)))

	if token != "" {
		req.Header.Set("Authorization", token)
	}

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

	if resp.StatusCode == http.StatusUnauthorized {
		c.accessToken = ""
		var refToken string
		_, refToken, err = c.token()
		if err != nil {
			return nil, err
		}
		c.r.SaveAndRestage(refToken)
		return nil, errors.New("unexpected status code 401")
	}

	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, data)
	}

	return data, nil
}

func (c *HTTPCurlClient) token() (string, string, error) {
	c.mu.RLock()
	accToken := c.accessToken
	c.mu.RUnlock()

	if accToken != "" {
		return accToken, "", nil
	}

	// We are unprotected via locks here, which can imply that multiple
	// go-routines are attempting to refresh the token at the same time. Both
	// will succeed and both will save the token. This is a performance
	// concern and unlikely. We are going to leave it for now.

	accessToken, refToken, err := c.f.Token()
	if err != nil {
		return "", "", err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = accessToken

	return c.accessToken, refToken, nil
}
