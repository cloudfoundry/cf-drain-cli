package uaa

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type UAATokenFetcher struct {
	d        Doer
	u        url.URL
	username string
	password string
}

func NewUAATokenFetcher(
	uaaAddr string,
	clientID string,
	clientSecret string,
	username string,
	password string,
	d Doer,
) *UAATokenFetcher {
	u, err := url.Parse(uaaAddr)
	if err != nil {
		panic(err)
	}
	u.User = url.UserPassword(clientID, clientSecret)

	return &UAATokenFetcher{
		d: d,

		// Take a copy so we can manipulate without race conditions
		u:        *u,
		username: username,
		password: password,
	}
}

func (f *UAATokenFetcher) Token() (string, error) {
	f.u.Path = "/oauth/token"
	req, _ := http.NewRequest(
		"POST",
		f.u.String(),
		ioutil.NopCloser(
			strings.NewReader(
				fmt.Sprintf("grant_type=password&response_type=token&username=%s&password=%s", f.username, f.password),
			),
		),
	)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.d.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get auth token, expected 200, got %d", resp.StatusCode)
	}

	var tokenOutput struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	err = json.NewDecoder(resp.Body).Decode(&tokenOutput)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s %s", tokenOutput.TokenType, tokenOutput.AccessToken), nil
}
