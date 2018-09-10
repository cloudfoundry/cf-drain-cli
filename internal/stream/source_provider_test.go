package stream_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"code.cloudfoundry.org/cf-drain-cli/internal/stream"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SingleOrSpaceProvider", func() {
	It("fetches all sourceIDs from a space", func() {
		httpClient := &stubHTTPClient{
			bodies:      []string{singleServiceInstancResponseBody, spaceAppsResponseBody},
			statusCodes: []int{http.StatusOK, http.StatusOK},
		}

		p := stream.NewSingleOrSpaceProvider(
			"", // Leaving this empty implies a space drain
			"http://localhost",
			"space-1",
			true,
			stream.WithSourceProviderClient(httpClient),
		)
		sIDs, err := p.SourceIDs()
		Expect(err).ToNot(HaveOccurred())
		Expect(sIDs).To(Equal([]string{"service-1", "app-1"}))
	})

	It("SourceIDs returns an error if fetching resources fails", func() {
		httpClient := &stubHTTPClient{
			bodies:      []string{singleServiceInstancResponseBody, spaceAppsResponseBody},
			statusCodes: []int{http.StatusOK, http.StatusInternalServerError},
		}

		p := stream.NewSingleOrSpaceProvider(
			"", // Leaving this empty implies a space drain
			"http://localhost",
			"space-1",
			true,
			stream.WithSourceProviderClient(httpClient),
		)
		_, err := p.SourceIDs()
		Expect(err).To(HaveOccurred())
	})

	It("fetches a single resource that is an app", func() {
		httpClient := &stubHTTPClient{
			bodies:      []string{appResponseBody},
			statusCodes: []int{http.StatusOK},
		}

		p := stream.NewSingleOrSpaceProvider(
			"app-1",
			"http://localhost",
			"space-1",
			false,
			stream.WithSourceProviderClient(httpClient),
		)
		r, err := p.Resources()
		Expect(err).ToNot(HaveOccurred())
		Expect(r).To(Equal([]stream.Resource{
			{GUID: "app-1", Name: "app-1-name"},
		}))

		Expect(httpClient.requestURLs).To(HaveLen(1))
		Expect(httpClient.requestURLs[0]).To(Equal("http://localhost/v3/apps/app-1"))
	})

	It("fetches a single resource that is a service instance", func() {
		httpClient := &stubHTTPClient{
			bodies:      []string{"{}", singleServiceInstancResponseBody},
			statusCodes: []int{http.StatusNotFound, http.StatusOK},
		}

		p := stream.NewSingleOrSpaceProvider(
			"service-1",
			"http://localhost",
			"space-1",
			false,
			stream.WithSourceProviderClient(httpClient),
		)

		r, err := p.Resources()
		Expect(err).ToNot(HaveOccurred())
		Expect(r).To(Equal([]stream.Resource{
			{GUID: "service-1", Name: "service-1-name"},
		}))

		Expect(httpClient.requestURLs).To(HaveLen(2))
		Expect(httpClient.requestURLs[1]).To(Equal("http://localhost/v3/service_instances?space_guids=space-1"))
	})

	It("fetches all the services and apps in a space", func() {
		httpClient := &stubHTTPClient{
			bodies:      []string{singleServiceInstancResponseBody, spaceAppsResponseBody},
			statusCodes: []int{http.StatusOK, http.StatusOK},
		}

		p := stream.NewSingleOrSpaceProvider(
			"", // Leaving this empty implies a space drain
			"http://localhost",
			"space-1",
			true,
			stream.WithSourceProviderClient(httpClient),
		)
		r, err := p.Resources()
		Expect(err).ToNot(HaveOccurred())
		Expect(r).To(Equal([]stream.Resource{
			{GUID: "service-1", Name: "service-1-name"},
			{GUID: "app-1", Name: "app-1-name"},
		}))

		Expect(httpClient.requestURLs).To(HaveLen(2))
		Expect(httpClient.requestURLs[0]).To(Equal("http://localhost/v3/service_instances?space_guids=space-1"))
		Expect(httpClient.requestURLs[1]).To(Equal("http://localhost/v3/apps?space_guids=space-1"))
	})

	It("returns an error if given invalid JSON", func() {
		httpClient := &stubHTTPClient{
			bodies:      []string{invalidResponseBody},
			statusCodes: []int{http.StatusOK},
		}

		p := stream.NewSingleOrSpaceProvider(
			"app-1",
			"http://localhost",
			"space-1",
			false,
			stream.WithSourceProviderClient(httpClient),
		)
		_, err := p.Resources()
		Expect(err).To(HaveOccurred())
	})

	It("returns the error when requesting the single app resource fails", func() {
		httpClient := &stubHTTPClient{
			bodies:      []string{},
			statusCodes: []int{},
			errors:      []error{errors.New("an error")},
		}

		p := stream.NewSingleOrSpaceProvider(
			"app-1",
			"http://localhost",
			"space-1",
			false,
			stream.WithSourceProviderClient(httpClient),
		)
		_, err := p.Resources()
		Expect(err).To(MatchError("an error"))
	})

	It("returns the error when requesting the service instances fails", func() {
		httpClient := &stubHTTPClient{
			bodies:      []string{"{}"},
			statusCodes: []int{http.StatusNotFound},
			errors:      []error{nil, errors.New("an error")},
		}

		p := stream.NewSingleOrSpaceProvider(
			"service-1",
			"http://localhost",
			"space-1",
			false,
			stream.WithSourceProviderClient(httpClient),
		)

		_, err := p.Resources()
		Expect(err).To(MatchError("an error"))
	})

	It("returns an error while fetching space info and gets invalid JSON for service instances", func() {
		httpClient := &stubHTTPClient{
			bodies:      []string{invalidResponseBody, spaceAppsResponseBody},
			statusCodes: []int{http.StatusOK, http.StatusOK},
		}

		p := stream.NewSingleOrSpaceProvider(
			"", // Leaving this empty implies a space drain
			"http://localhost",
			"space-1",
			true,
			stream.WithSourceProviderClient(httpClient),
		)
		_, err := p.Resources()
		Expect(err).To(HaveOccurred())
	})

	It("returns an error while fetching space info and gets invalid JSON for apps", func() {
		httpClient := &stubHTTPClient{
			bodies:      []string{singleServiceInstancResponseBody, invalidResponseBody},
			statusCodes: []int{http.StatusOK, http.StatusOK},
		}

		p := stream.NewSingleOrSpaceProvider(
			"", // Leaving this empty implies a space drain
			"http://localhost",
			"space-1",
			true,
			stream.WithSourceProviderClient(httpClient),
		)
		_, err := p.Resources()
		Expect(err).To(HaveOccurred())
	})
})

type stubHTTPClient struct {
	bodies      []string
	statusCodes []int
	errors      []error
	requestURLs []string

	requestCount int
}

func (c *stubHTTPClient) Get(url string) (*http.Response, error) {
	defer func() {
		c.requestCount++
	}()

	c.requestURLs = append(c.requestURLs, url)

	var err error
	if len(c.errors) > c.requestCount {
		err = c.errors[c.requestCount]
	}

	if err != nil {
		return nil, err
	}

	resp := &http.Response{
		StatusCode: c.statusCodes[c.requestCount],
		Body:       ioutil.NopCloser(strings.NewReader(c.bodies[c.requestCount])),
	}

	return resp, nil
}

var (
	appResponseBody = `{
		"guid": "app-1",
		"name": "app-1-name"
	}`

	spaceAppsResponseBody = `{
		"resources": [
			{
				"guid": "app-1",
				"name": "app-1-name"
			}
		]
	}`

	singleServiceInstancResponseBody = `{
		"resources": [
			{
				"guid": "service-1",
				"name": "service-1-name"
			}
		]
	}`

	invalidResponseBody = `invalid`
)
