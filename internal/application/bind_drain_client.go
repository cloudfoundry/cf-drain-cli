package application

// BindDrainClient is a noop bind drain client. Currently, there is no need
// of a bind step. When we push the syslog_forwarder it is already "bound" to
// the app. We spoke with product and a bind step will likely be required in
// the near future.
type BindDrainClient struct{}

func NewBindDrainClient() *BindDrainClient {
	return &BindDrainClient{}
}

func (c *BindDrainClient) BindDrain(appGuid, serviceInstanceGuid string) error {
	return nil
}
