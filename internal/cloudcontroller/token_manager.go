package cloudcontroller

import (
	"fmt"
	"net/http"
)

type AuthCurler interface {
	AuthCurl(url, method, body, token string) ([]byte, error)
}

type UAAClient interface {
	GetRefreshToken(clientID, refreshToken string, insecureSkipVerify bool) (string, string, error)
}

type Logger interface {
	Fatalf(format string, v ...interface{})
}

type TokenManager struct {
	c                  AuthCurler
	uaa                UAAClient
	clientID           string
	refreshToken       string
	appGUID            string
	insecureSkipVerify bool
	log                Logger
}

func NewTokenManager(
	c AuthCurler,
	uaa UAAClient,
	clientID string,
	initialRefreshToken string,
	appGUID string,
	skipCertVerify bool,
	log Logger,

) *TokenManager {
	return &TokenManager{
		c:                  c,
		uaa:                uaa,
		clientID:           clientID,
		refreshToken:       initialRefreshToken,
		appGUID:            appGUID,
		insecureSkipVerify: skipCertVerify,
		log:                log,
	}
}

func (m *TokenManager) Token() (string, string, error) {
	refToken, accToken, err := m.uaa.GetRefreshToken(m.clientID, m.refreshToken, m.insecureSkipVerify)
	if err != nil {
		m.log.Fatalf("Failed to fetch tokens from UAA: %s", err)
	}

	m.saveRefreshToken(accToken, refToken)
	return accToken, refToken, nil
}

func (m *TokenManager) saveRefreshToken(accessToken, refreshToken string) {
	url := fmt.Sprintf("/v3/apps/%s/environment_variables", m.appGUID)
	body := fmt.Sprintf(`{"var":{"REFRESH_TOKEN": %q}}`, refreshToken)
	_, err := m.c.AuthCurl(url, http.MethodPatch, body, accessToken)
	if err != nil {
		m.log.Fatalf("Failed to updated REFRESH_TOKEN with cloud controller: %s", err)
	}

	m.refreshToken = refreshToken
}
