package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type SingleOrSpaceProvider struct {
	Source    string
	ApiAddr   string
	SpaceGuid string
}

func (s *SingleOrSpaceProvider) SourceIDs() ([]string, error) {
	if s.Source != "" {
		return []string{s.Source}, nil
	}

	return serviceInstanceGuids(s.ApiAddr, s.SpaceGuid)
}

func serviceInstanceGuids(api, space string) ([]string, error) {
	resp, err := http.Get(api + "/v3/service_instances?space_guids=" + space)
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
