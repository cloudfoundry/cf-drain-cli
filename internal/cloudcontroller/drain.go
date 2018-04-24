package cloudcontroller

type Drain struct {
	Name     string
	Guid     string
	Apps     []string
	AppGuids []string
	Type     string
	DrainURL string
}
