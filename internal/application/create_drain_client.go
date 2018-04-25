package application

import "fmt"

type CreateDrainClient struct{}

func NewCreateDrainClient(c Curler) *CreateDrainClient {
	return &CreateDrainClient{}
}

func (c *CreateDrainClient) CreateDrain(name, url, spaceGuid, drainType string) error {
	if !validDrainType(drainType) {
		return fmt.Errorf("invalid drain type: %s", drainType)
	}

	// push the app with nostart
	// set env vars
	// start the app

	panic("not implemented")
	return nil
}

func validDrainType(drainType string) bool {
	switch drainType {
	case "all", "metrics", "logs":
		return true
	default:
		return false
	}
}
