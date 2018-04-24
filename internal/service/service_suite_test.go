package service_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Suite")
}

type stubCurler struct {
	URLs    []string
	methods []string
	bodies  []string
	resps   map[string]string
	errs    map[string]error
}

func newStubCurler() *stubCurler {
	return &stubCurler{
		resps: make(map[string]string),
		errs:  make(map[string]error),
	}
}

func (s *stubCurler) Curl(URL, method, body string) ([]byte, error) {
	s.URLs = append(s.URLs, URL)
	s.methods = append(s.methods, method)
	s.bodies = append(s.bodies, body)
	return []byte(s.resps[URL]), s.errs[URL]
}
