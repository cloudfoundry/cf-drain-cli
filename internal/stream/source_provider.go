package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type SingleOrSpaceProvider struct {
	Source          string
	ApiAddr         string
	SpaceGuid       string
	IncludeServices bool
}

func (s *SingleOrSpaceProvider) SourceIDs() ([]string, error) {
	if s.Source != "" {
		return []string{s.Source}, nil
	}

	sg, err := s.serviceInstanceGuids()
	if err != nil {
		return nil, err
	}

	ag, err := s.appGuids()
	if err != nil {
		return nil, err
	}

	return append(sg, ag...), nil
}

func (s *SingleOrSpaceProvider) appGuids() ([]string, error) {
	return s.resourceGuids("apps")
}

func (s *SingleOrSpaceProvider) serviceInstanceGuids() ([]string, error) {
	if s.IncludeServices {
		return s.resourceGuids("service_instances")
	}

	return nil, nil
}

func (s *SingleOrSpaceProvider) resourceGuids(resource string) ([]string, error) {
	url := fmt.Sprintf("%s/v3/%s?space_guids=%s", s.ApiAddr, resource, s.SpaceGuid)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("failed to make capi request: %s", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected status code from cc api: %d", resp.StatusCode)
		return nil, errors.New(fmt.Sprintf("unexpected status code from cc api: %d", resp.StatusCode))
	}

	var sir struct {
		Resources []struct {
			GUID string `json:"guid"`
		} `json:"resources"`
	}

	err = json.NewDecoder(resp.Body).Decode(&sir)
	if err != nil {
		log.Fatalf("could not unmarshal from cc api: %s", err)
		return nil, err
	}

	var guids []string
	for _, r := range sir.Resources {
		guids = append(guids, r.GUID)
	}

	return guids, nil
}
