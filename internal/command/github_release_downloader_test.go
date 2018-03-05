package command_test

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GithubReleaseDownloader", func() {
	var (
		httpClient *spyHTTPClient
		logger     *stubLogger
		d          command.GithubReleaseDownloader
	)

	BeforeEach(func() {
		httpClient = newSpyHTTPClient()
		logger = &stubLogger{}
		d = command.NewGithubReleaseDownloader(httpClient, logger)
	})

	It("returns a directory path to the latest release", func() {
		httpClient.m["https://api.github.com/repos/cloudfoundry/cf-drain-cli/releases"] = httpResponse{
			r: &http.Response{
				StatusCode: 200,
				Body:       releasesResponse(),
			},
		}

		httpClient.m["https://github.com/cloudfoundry/cf-drain-cli/releases/download/v0.5/space_drain"] = httpResponse{
			r: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(`Github File`)),
			},
		}

		p := d.Download()
		Expect(path.Base(p)).To(Equal("space_drain"))

		file, err := os.Open(p)
		Expect(err).ToNot(HaveOccurred())

		contents, err := ioutil.ReadAll(file)
		Expect(err).ToNot(HaveOccurred())

		Expect(string(contents)).To(Equal("Github File"))

		info, err := file.Stat()
		Expect(err).ToNot(HaveOccurred())
		Expect(int(info.Mode() & 0111)).To(Equal(0111))
	})

	It("fatally logs when fetching releases returns a non-200", func() {
		httpClient.m["https://api.github.com/repos/cloudfoundry/cf-drain-cli/releases"] = httpResponse{
			r: &http.Response{StatusCode: 404},
		}

		Expect(func() {
			d.Download()
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("unexpected status code (404) from github"))
	})

	It("fatally logs when fetching the latest asset returns a non-200", func() {
		httpClient.m["https://api.github.com/repos/cloudfoundry/cf-drain-cli/releases"] = httpResponse{
			r: &http.Response{
				StatusCode: 200,
				Body:       releasesResponse(),
			},
		}

		httpClient.m["https://github.com/cloudfoundry/cf-drain-cli/releases/download/v0.5/space_drain"] = httpResponse{
			r: &http.Response{StatusCode: 404},
		}

		Expect(func() {
			d.Download()
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("unexpected status code (404) from github"))
	})

	It("fatally logs when it can't find the space drain", func() {
		httpClient.m["https://api.github.com/repos/cloudfoundry/cf-drain-cli/releases"] = httpResponse{
			r: &http.Response{
				StatusCode: 200,
				Body:       releasesResponseNoSpaceDrain(),
			},
		}

		Expect(func() {
			d.Download()
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("unable to find space_drain asset in releases"))
	})

	It("fatally logs when decoding releases fails", func() {
		httpClient.m["https://api.github.com/repos/cloudfoundry/cf-drain-cli/releases"] = httpResponse{
			r: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("invalid")),
			},
		}

		Expect(func() {
			d.Download()
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("failed to decode releases response from github"))
	})

	It("fatally logs when github returns an error", func() {
		httpClient.m["https://api.github.com/repos/cloudfoundry/cf-drain-cli/releases"] = httpResponse{
			err: errors.New("some error"),
		}

		Expect(func() {
			d.Download()
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("failed to read from github: some error"))
	})
})

func releasesResponse() io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(`
   [
     {
      "tag_name": "v0.4.1",
      "assets": [
        {
          "name": "something",
          "browser_download_url": "https://github.com/cloudfoundry/cf-drain-cli/releases/download/v0.4.1/something"
        },
        {
          "name": "space_drain",
          "browser_download_url": "https://github.com/cloudfoundry/cf-drain-cli/releases/download/v0.4.1/space_drain"
        }
      ]
     },
     {
      "tag_name": "v0.5",
      "assets": [
        {
          "name": "something",
          "browser_download_url": "https://github.com/cloudfoundry/cf-drain-cli/releases/download/v0.5/something"
        },
        {
          "name": "space_drain",
          "browser_download_url": "https://github.com/cloudfoundry/cf-drain-cli/releases/download/v0.5/space_drain"
        }
      ]
     }
   ]
`))
}

func releasesResponseNoSpaceDrain() io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(`
   [
     {
      "tag_name": "v0.5",
      "assets": [
        {
          "name": "something",
          "browser_download_url": "https://github.com/cloudfoundry/cf-drain-cli/releases/download/v0.5/something"
        }
      ]
     }
   ]
`))
}

type httpResponse struct {
	r   *http.Response
	err error
}

type spyHTTPClient struct {
	m map[string]httpResponse
}

func newSpyHTTPClient() *spyHTTPClient {
	return &spyHTTPClient{
		m: make(map[string]httpResponse),
	}
}

func (s *spyHTTPClient) Do(r *http.Request) (*http.Response, error) {
	if r.Method != http.MethodGet {
		panic("only use GETs")
	}

	value, ok := s.m[r.URL.String()]
	if !ok {
		panic("unknown URL " + r.URL.String())
	}

	return value.r, value.err
}
