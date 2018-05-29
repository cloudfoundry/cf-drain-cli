package cloudcontroller_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var _ = Describe("TokenManager", func() {
	var (
		uaa        *spyUAAClient
		m          *cloudcontroller.TokenManager
		stubLogger *stubLogger
	)

	BeforeEach(func() {
		uaa = &spyUAAClient{}
		uaa.respAccessToken = "access-token"
		uaa.respRefreshToken = "new-refresh-token"
		stubLogger = newStubLogger()

		m = cloudcontroller.NewTokenManager(
			uaa,
			"client-id",
			"initial-refresh-token",
			"app-guid",
			false,
			stubLogger,
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

	It("sets insecureSkipVerify", func() {
		m = cloudcontroller.NewTokenManager(
			uaa,
			"client-id",
			"refresh-token",
			"appguid",
			true,
			stubLogger,
		)
		m.Token()
		Expect(uaa.reqSkipCertVerify).To(BeTrue())

		m = cloudcontroller.NewTokenManager(
			uaa,
			"client-id",
			"refresh-token",
			"appguid",
			false,
			stubLogger,
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
		Expect(stubLogger.called).To(Equal(1))

		// recovery
		uaa.respError = nil
		m.Token()

		Expect(uaa.reqRefreshToken).To(Equal("initial-refresh-token"))
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
	urls    []string
	methods []string
	bodies  []string
	errs    []error
}

func (s *spyAuthCurler) Curl(url string, method string, body string) ([]byte, error) {
	s.urls = append(s.urls, url)
	s.methods = append(s.methods, method)
	s.bodies = append(s.bodies, body)

	if len(s.errs) == 0 {
		return nil, nil
	}

	e := s.errs[0]
	s.errs = s.errs[1:]

	return nil, e
}

type stubLogger struct {
	called int
}

func newStubLogger() *stubLogger {
	return &stubLogger{}
}

func (s *stubLogger) Fatalf(format string, v ...interface{}) {
	s.called++
	panic("fatal")
}
