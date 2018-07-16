package groupmanager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Curler interface {
	Get(url string) (*http.Response, error)
}

type SpaceProvider struct {
	c         Curler
	spaceGuid string
	apiURL    string
}

func Space(c Curler, apiURL, spaceGuid string) *SpaceProvider {
	sp := &SpaceProvider{
		c:         c,
		spaceGuid: spaceGuid,
		apiURL:    apiURL,
	}

	return sp
}

func (s *SpaceProvider) SourceIDs() []string {
	return s.guidsFor("service_instances")
}

func (s *SpaceProvider) guidsFor(resource string) []string {
	resp, err := s.c.Get(fmt.Sprintf("%s/v3/%s?space_guids=%s", s.apiURL, resource, s.spaceGuid))
	if err != nil {
		log.Printf("error getting app info from CAPI: %s", err)
		return nil
	}

	var capiResources response
	err = json.NewDecoder(resp.Body).Decode(&capiResources)
	if err != nil {
		log.Printf("error getting app info from CAPI: %s", err)
		return nil
	}

	var guids []string
	for _, resource := range capiResources.Resources {
		guids = append(guids, resource.Guid)
	}
	return guids
}

type response struct {
	Resources []struct {
		Guid string `json:"guid"`
	} `json:"resources"`
}
