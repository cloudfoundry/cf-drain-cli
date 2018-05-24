package cloudcontroller_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var _ = Describe("HttpCurlClient", func() {
	var (
		doer    *spyDoer
		fetcher *stubTokenFetcher
		c       *cloudcontroller.HTTPCurlClient
	)

	BeforeEach(func() {
		doer = newSpyDoer()
		fetcher = newStubTokenFetcher()
		c = cloudcontroller.NewHTTPCurlClient("https://api.system-domain.com", doer, fetcher)
	})

	It("hits the correct URL", func() {
		doer.respBody = "resp-body"
		body, err := c.Curl("/v2/some-url", "PUT", "some-body")
		Expect(err).ToNot(HaveOccurred())

		Expect(doer.URLs).To(ConsistOf("https://api.system-domain.com/v2/some-url"))
		Expect(doer.methods).To(ConsistOf("PUT"))
		Expect(string(body)).To(Equal("resp-body"))
		Expect(doer.bodies).To(ConsistOf("some-body"))
	})

	It("populates Authorization header", func() {
		fetcher.tokens = []string{"some-token"}
		fetcher.errs = []error{nil}

		_, err := c.Curl("some-url", "PUT", "some-body")
		Expect(err).ToNot(HaveOccurred())

		Expect(doer.headers).To(ConsistOf(HaveKeyWithValue("Authorization", []string{"some-token"})))
	})

	It("returns an error if the TokenFetcher fails", func() {
		fetcher.tokens = []string{""}
		fetcher.errs = []error{errors.New("token fetch failure")}

		_, err := c.Curl("some-url", "PUT", "some-body")

		Expect(err).To(HaveOccurred())
		Expect(doer.URLs).To(BeEmpty())
	})

	It("returns error for non-2XX status code", func() {
		doer.statusCode = 404

		_, err := c.Curl("some-url", "PUT", "some-body")
		Expect(err).To(HaveOccurred())
	})

	It("returns error if Doer fails", func() {
		doer.err = errors.New("some-error")

		_, err := c.Curl("some-url", "PUT", "some-body")
		Expect(err).To(MatchError("some-error"))
	})

	It("attaches the header 'Content-Type' for non-GET requests", func() {
		c.Curl("some-url", "GET", "")
		Expect(doer.headers).ToNot(
			ContainElement(HaveKeyWithValue("Content-Type", []string{"application/json"})),
		)

		c.Curl("some-url", "PUT", "some-body")
		Expect(doer.headers).To(
			ContainElement(HaveKeyWithValue("Content-Type", []string{"application/json"})),
		)
	})

	It("panics if method is GET and has a body", func() {
		Expect(func() {
			c.Curl("some-url", "GET", "some-body")
		}).To(Panic())
	})

	It("hits the correct URL and populates the Authorization header", func() {
		doer.respBody = "resp-body"
		body, err := c.AuthCurl("/v2/some-url", "PUT", "some-body", "some-token")
		Expect(err).ToNot(HaveOccurred())

		Expect(doer.URLs).To(ConsistOf("https://api.system-domain.com/v2/some-url"))
		Expect(doer.methods).To(ConsistOf("PUT"))
		Expect(string(body)).To(Equal("resp-body"))
		Expect(doer.bodies).To(ConsistOf("some-body"))

		Expect(doer.headers).To(ConsistOf(HaveKeyWithValue("Authorization", []string{"some-token"})))
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

type stubTokenFetcher struct {
	tokens []string
	errs   []error
}

func newStubTokenFetcher() *stubTokenFetcher {
	return &stubTokenFetcher{}
}

func (s *stubTokenFetcher) Token() (string, string, error) {
	if len(s.tokens) != len(s.errs) {
		panic("tokens and errs are out of sync")
	}

	if len(s.tokens) == 0 {
		return "", "", nil
	}

	t := s.tokens[0]
	s.tokens = s.tokens[1:]

	e := s.errs[0]
	s.errs = s.errs[1:]

	return t, "", e
}
