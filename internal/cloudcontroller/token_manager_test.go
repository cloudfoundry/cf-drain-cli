package cloudcontroller_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var _ = Describe("TokenManager", func() {
	var (
		uaa *spyUAAClient
		ac  *spyAuthCurler
		m   *cloudcontroller.TokenManager
	)

	BeforeEach(func() {
		uaa = &spyUAAClient{}
		uaa.respAccessToken = "access-token"
		uaa.respRefreshToken = "new-refresh-token"

		ac = &spyAuthCurler{}
		m = cloudcontroller.NewTokenManager(
			ac,
			uaa,
			"client-id",
			"initial-refresh-token",
			"app-guid",
			false,
		)
	})

	It("uses a refresh token to fetch an access token", func() {
		token, refToken, err := m.Token()
		Expect(err).ToNot(HaveOccurred())

		Expect(token).To(Equal("access-token"))
		Expect(refToken).To(Equal("new-refresh-token"))

		Expect(uaa.reqRefreshToken).To(Equal("initial-refresh-token"))
		Expect(uaa.reqClientID).To(Equal("client-id"))
	})

	It("uses the AuthCurler to curl CAPI", func() {
		_, refresh, _ := m.Token()
		Expect(uaa.reqRefreshToken).To(Equal("initial-refresh-token"))
		Expect(refresh).To(Equal("new-refresh-token"))

		Expect(ac.url).To(Equal("/v3/apps/app-guid/environment_variables"))
		Expect(ac.method).To(Equal("PATCH"))
		Expect(ac.token).To(Equal("access-token"))
		Expect(ac.body).To(MatchJSON(`{
			     "var": {
					 "REFRESH_TOKEN": "new-refresh-token"
				}
		}`))

		m.Token()
		Expect(uaa.reqRefreshToken).To(Equal("new-refresh-token"))
	})

	It("sets insecureSkipVerify", func() {
		m = cloudcontroller.NewTokenManager(
			ac,
			uaa,
			"client-id",
			"refresh-token",
			"appguid",
			true,
		)
		m.Token()
		Expect(uaa.reqSkipCertVerify).To(BeTrue())

		m = cloudcontroller.NewTokenManager(
			ac,
			uaa,
			"client-id",
			"refresh-token",
			"appguid",
			false,
		)
		m.Token()
		Expect(uaa.reqSkipCertVerify).To(BeFalse())
	})

	It("returns an error if UAA fails", func() {
		uaa.respError = errors.New("uaa-error")
		Expect(func() { m.Token() }).To(Panic())
	})

	It("does not overwrite the refresh token if GetRefreshToken fails", func() {
		uaa.respError = errors.New("Failed to fetch tokens from UAA")
		Expect(func() { m.Token() }).To(Panic())

		// recovery
		uaa.respError = nil
		m.Token()

		Expect(uaa.reqRefreshToken).To(Equal("initial-refresh-token"))
	})

	It("panics if unable to save REFRESH_TOKEN to cloud controller", func() {
		ac.err = errors.New("CAPI is down")
		Expect(func() { m.Token() }).To(Panic())
	})
})

type spyUAAClient struct {
	reqClientID       string
	reqRefreshToken   string
	reqSkipCertVerify bool

	respRefreshToken string
	respAccessToken  string
	respError        error
}

func (s *spyUAAClient) GetRefreshToken(clientID, refreshToken string, insecureSkipVerify bool) (string, string, error) {
	s.reqClientID = clientID
	s.reqRefreshToken = refreshToken
	s.reqSkipCertVerify = insecureSkipVerify

	return s.respRefreshToken, s.respAccessToken, s.respError
}

type spyAuthCurler struct {
	url    string
	method string
	body   string
	token  string
	err    error
}

func (s *spyAuthCurler) AuthCurl(url string, method string, body string, token string) ([]byte, error) {
	s.url = url
	s.method = method
	s.body = body
	s.token = token

	return nil, s.err
}
