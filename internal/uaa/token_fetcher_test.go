package uaa_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/uaa"
)

var _ = Describe("UaaTokenFetcher", func() {
	var (
		doer *spyDoer
		f    *uaa.UAATokenFetcher
	)

	BeforeEach(func() {
		doer = newSpyDoer()
		doer.respBody = `{"access_token": "some-token", "token_type": "bearer"}`
		f = uaa.NewUAATokenFetcher(
			"https://uaa.system-domain.com",
			"some-id",
			"some-secret",
			"some-user",
			"some-password",
			doer,
		)
	})

	It("requests a token from the UAA", func() {
		token, err := f.Token()

		Expect(err).ToNot(HaveOccurred())
		Expect(token).To(Equal("bearer some-token"))
		Expect(doer.URLs).To(ConsistOf("https://some-id:some-secret@uaa.system-domain.com/oauth/token"))
		Expect(doer.methods).To(ConsistOf("POST"))
		Expect(doer.headers).To(ConsistOf(
			And(
				HaveKeyWithValue("Accept", []string{"application/json"}),
				HaveKeyWithValue("Content-Type", []string{"application/x-www-form-urlencoded"}),
			),
		))

		Expect(doer.bodies).To(ConsistOf(
			"grant_type=password&response_type=token&username=some-user&password=some-password",
		))
	})

	It("returns an error if UAA fails", func() {
		doer.err = errors.New("some-uaa-error")
		_, err := f.Token()
		Expect(err).To(MatchError("some-uaa-error"))
	})

	It("returns an error if it fails to decode JSON", func() {
		doer.respBody = `invalid`
		_, err := f.Token()
		Expect(err).To(HaveOccurred())
	})
})

type spyDoer struct {
	URLs    []string
	bodies  []string
	methods []string
	headers []http.Header
	users   []*url.Userinfo

	statusCode int
	err        error
	respBody   string
}

func newSpyDoer() *spyDoer {
	return &spyDoer{
		statusCode: 200,
	}
}

func (s *spyDoer) Do(r *http.Request) (*http.Response, error) {
	s.URLs = append(s.URLs, r.URL.String())
	s.methods = append(s.methods, r.Method)
	s.headers = append(s.headers, r.Header)
	s.users = append(s.users, r.URL.User)

	var body []byte
	if r.Body != nil {
		var err error
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
	}

	s.bodies = append(s.bodies, string(body))

	return &http.Response{
		StatusCode: s.statusCode,
		Body:       ioutil.NopCloser(strings.NewReader(s.respBody)),
	}, s.err
}
