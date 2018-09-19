package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type SingleOrSpaceProvider struct {
	// TODO: Stop exporting these

	Source          string
	ApiAddr         string
	SpaceGuid       string
	IncludeServices bool
	httpClient      Getter
	excludeFilter   SourceIDFilter
}

func NewSingleOrSpaceProvider(
	sourceID string,
	apiAddr string,
	spaceID string,
	includeServices bool,
	opts ...SingleOrSpaceProviderOption,
) *SingleOrSpaceProvider {
	ssp := &SingleOrSpaceProvider{
		Source:          sourceID,
		ApiAddr:         apiAddr,
		SpaceGuid:       spaceID,
		IncludeServices: includeServices,
		httpClient:      http.DefaultClient,
		excludeFilter:   func(string) bool { return false },
	}

	for _, o := range opts {
		o(ssp)
	}

	return ssp
}

func (s *SingleOrSpaceProvider) Resources() ([]Resource, error) {
	if s.Source != "" {
		return s.resourcesForSingleApp()
	}
	return s.resourcesForSpace()
}

func (s *SingleOrSpaceProvider) resourcesForSingleApp() ([]Resource, error) {
	url := fmt.Sprintf("%s/v3/apps/%s", s.ApiAddr, s.Source)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resources, err := s.resources("service_instances")
		if err != nil {
			return nil, err
		}

		for _, resource := range resources {
			if resource.GUID == s.Source {
				return []Resource{resource}, nil
			}
		}
	}

	var resource Resource
	err = json.NewDecoder(resp.Body).Decode(&resource)
	if err != nil {
		return nil, err
	}
	return []Resource{resource}, nil
}

func (s *SingleOrSpaceProvider) resourcesForSpace() ([]Resource, error) {
	sg, err := s.serviceInstances()
	if err != nil {
		return nil, err
	}

	ag, err := s.apps()
	if err != nil {
		return nil, err
	}

	resources := append(sg, ag...)

	var filtered []Resource
	for _, r := range resources {
		if !s.excludeFilter(r.GUID) {
			filtered = append(filtered, r)
		}
	}

	return filtered, nil
}

func (s *SingleOrSpaceProvider) apps() ([]Resource, error) {
	return s.resources("apps")
}

func (s *SingleOrSpaceProvider) serviceInstances() ([]Resource, error) {
	if s.IncludeServices {
		return s.resources("service_instances")
	}

	return nil, nil
}

func (s *SingleOrSpaceProvider) resources(resource string) ([]Resource, error) {
	url := fmt.Sprintf("%s/v3/%s?space_guids=%s", s.ApiAddr, resource, s.SpaceGuid)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		log.Printf("failed to make capi request: %s", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected status code from cc api: %d", resp.StatusCode)
		return nil, errors.New(fmt.Sprintf("unexpected status code from cc api: %d", resp.StatusCode))
	}

	var sir struct {
		Resources []Resource `json:"resources"`
	}

	err = json.NewDecoder(resp.Body).Decode(&sir)
	if err != nil {
		return nil, err
	}

	return sir.Resources, nil
}

type SingleOrSpaceProviderOption func(*SingleOrSpaceProvider)

func WithSourceProviderClient(httpClient Getter) SingleOrSpaceProviderOption {
	return func(ssp *SingleOrSpaceProvider) {
		ssp.httpClient = httpClient
	}
}

// SourceIDFilter returns true if the passed source id should be filtered from
// the space list from CAPI
type SourceIDFilter func(string) bool

func WithSourceProviderSpaceExcludeFilter(excludeFilter SourceIDFilter) SingleOrSpaceProviderOption {
	return func(ssp *SingleOrSpaceProvider) {
		ssp.excludeFilter = excludeFilter
	}
}

type Resource struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}

type Getter interface {
	Get(url string) (*http.Response, error)
}
