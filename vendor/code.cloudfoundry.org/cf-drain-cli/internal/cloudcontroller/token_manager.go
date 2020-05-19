package cloudcontroller

type AuthCurler interface {
	Curl(url, method, body string) ([]byte, error)
}

type UAAClient interface {
	GetRefreshToken(clientID, refreshToken string, insecureSkipVerify bool) (string, string, error)
}

type Logger interface {
	Fatalf(format string, v ...interface{})
}

type TokenManager struct {
	uaa                UAAClient
	clientID           string
	refreshToken       string
	appGUID            string
	insecureSkipVerify bool
	log                Logger
}

func NewTokenManager(
	uaa UAAClient,
	clientID string,
	initialRefreshToken string,
	appGUID string,
	skipCertVerify bool,
	log Logger,

) *TokenManager {
	return &TokenManager{
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
	m.refreshToken = refToken

	return accToken, refToken, nil
}
