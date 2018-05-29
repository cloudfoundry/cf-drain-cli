package cloudcontroller_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var _ = Describe("HttpCurlClient", func() {
	var (
		doer     *spyDoer
		fetcher  *spyTokenFetcher
		restager *spySaveAndRestager
		c        *cloudcontroller.HTTPCurlClient
	)

	BeforeEach(func() {
		doer = newSpyDoer()
		fetcher = newSpyTokenFetcher()
		restager = newSpySaveAndRestager()
		c = cloudcontroller.NewHTTPCurlClient("https://api.system-domain.com", doer, fetcher, restager)
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
		fetcher.refTokens = []string{"some-token"}
		fetcher.errs = []error{nil}

		_, err := c.Curl("some-url", "PUT", "some-body")
		Expect(err).ToNot(HaveOccurred())

		Expect(doer.headers).To(ContainElement(HaveKeyWithValue("Authorization", []string{"some-token"})))
	})

	It("reuses tokens until a 401", func() {
		fetcher.tokens = []string{"some-token", "some-other-token"}
		fetcher.refTokens = []string{"some-ref-token", "some-other-ref-token"}
		fetcher.errs = []error{nil, nil}

		_, err := c.Curl("some-url", "PUT", "some-body")
		Expect(err).ToNot(HaveOccurred())

		_, err = c.Curl("some-url", "PUT", "some-body")
		Expect(err).ToNot(HaveOccurred())

		Expect(fetcher.called).To(Equal(1))

		// Now go get a new token
		doer.statusCode = http.StatusUnauthorized
		doer.headers = nil
		c.Curl("some-url", "PUT", "some-body")
		Expect(fetcher.called).To(Equal(2))

		Expect(restager.refreshToken).To(Equal("some-other-ref-token"))
	})

	It("returns an error if the TokenFetcher fails", func() {
		fetcher.tokens = []string{""}
		fetcher.refTokens = []string{""}
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

	It("survives the race detector", func() {
		go func() {
			for i := 0; i < 100; i++ {
				c.Curl("/v2/some-url", "PUT", "some-body")
			}
		}()

		for i := 0; i < 100; i++ {
			c.Curl("/v2/some-url", "PUT", "some-body")
		}
	})
})

type spyDoer struct {
	mu      sync.Mutex
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
	s.mu.Lock()
	defer s.mu.Unlock()

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

type spyTokenFetcher struct {
	mu sync.Mutex

	called int

	tokens    []string
	refTokens []string
	errs      []error
}

func newSpyTokenFetcher() *spyTokenFetcher {
	return &spyTokenFetcher{}
}

func (s *spyTokenFetcher) Token() (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.called++

	if len(s.tokens) != len(s.errs) || len(s.tokens) != len(s.refTokens) {
		panic("tokens and errs are out of sync")
	}

	if len(s.tokens) == 0 {
		return "", "", nil
	}

	t := s.tokens[0]
	s.tokens = s.tokens[1:]

	r := s.refTokens[0]
	s.refTokens = s.refTokens[1:]

	e := s.errs[0]
	s.errs = s.errs[1:]

	return t, r, e
}

type spySaveAndRestager struct {
	mu           sync.Mutex
	refreshToken string
}

func newSpySaveAndRestager() *spySaveAndRestager {
	return &spySaveAndRestager{}
}

func (s *spySaveAndRestager) SaveAndRestage(refreshToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.refreshToken = refreshToken
}
