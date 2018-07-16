package groupmanager_test

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/cf-drain-cli/internal/groupmanager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SpaceProvider", func() {

	It("gets services for the space", func() {
		curler := &stubCurler{
			resp: [][]byte{
				[]byte(`{
					"resources": [
						{"guid": "service-1"},
						{"guid": "service-2"},
						{"guid": "service-3"}
					]
				}`),
			},
		}

		provider := groupmanager.Space(
			curler,
			"http://hostname.com",
			"space-guid",
		)

		Expect(provider.SourceIDs()).To(ConsistOf(
			"service-1",
			"service-2",
			"service-3",
		))

		Expect(curler.requestedURLs).To(Equal([]string{
			"http://hostname.com/v3/service_instances?space_guids=space-guid",
		}))
	})
})

type stubCurler struct {
	requestCount  int
	resp          [][]byte
	requestedURLs []string
}

func (s *stubCurler) Get(url string) (*http.Response, error) {
	s.requestedURLs = append(s.requestedURLs, url)
	resp := s.resp[s.requestCount]
	s.requestCount++

	return &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(resp)),
	}, nil
}
