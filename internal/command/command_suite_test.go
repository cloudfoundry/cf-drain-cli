package command_test

import (
	"fmt"
	"strings"
	"testing"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Command Suite")
}

type stubCliConnection struct {
	plugin.CliConnection

	sslDisabled bool

	getAppName  string
	getAppGuid  string
	getAppError error

	getServicesName  string
	getServicesError error
	getServicesApps  []string

	getServiceName  string
	getServiceGuid  string
	getServiceError error

	cliCommandWithoutTerminalOutputArgs     [][]string
	cliCommandWithoutTerminalOutputResponse map[string]string

	cliCommandArgs     [][]string
	createUserError    error
	createServiceError error
	bindServiceError   error
	unbindServiceError error
	deleteServiceError error
	pushAppError       error
	startAppError      error
	deleteAppError     error

	currentSpaceName  string
	currentSpaceGuid  string
	currentSpaceError error
	currentOrgName    string
	currentOrgError   error

	apiEndpoint      string
	apiEndpointError error

	setEnvErrors map[string]error
}

func newStubCliConnection() *stubCliConnection {
	return &stubCliConnection{
		cliCommandWithoutTerminalOutputResponse: make(map[string]string),
		setEnvErrors:                            make(map[string]error),
	}
}

func (s *stubCliConnection) GetApp(name string) (plugin_models.GetAppModel, error) {
	if s.getAppError == nil {
		s.getAppName = name
	}
	return plugin_models.GetAppModel{
		Name: s.getAppName,
		Guid: s.getAppGuid,
	}, s.getAppError
}

func (s *stubCliConnection) GetCurrentSpace() (plugin_models.Space, error) {
	return plugin_models.Space{
		plugin_models.SpaceFields{
			Name: s.currentSpaceName,
			Guid: s.currentSpaceGuid,
		},
	}, s.currentSpaceError
}

func (s *stubCliConnection) GetCurrentOrg() (plugin_models.Organization, error) {
	return plugin_models.Organization{
		plugin_models.OrganizationFields{
			Name: s.currentOrgName,
		},
	}, s.currentOrgError
}

func (s *stubCliConnection) GetServices() ([]plugin_models.GetServices_Model, error) {
	resp := []plugin_models.GetServices_Model{
		{
			Name:             "garbage-1",
			ApplicationNames: []string{"garbage-app-1", "garbage-app-2"},
		},
		{
			Name:             s.getServicesName,
			ApplicationNames: s.getServicesApps,
		},
		{
			Name:             "garbage-3",
			ApplicationNames: []string{"garbage-app-3", "garbage-app-4"},
		},
	}

	return resp, s.getServicesError
}

func (s *stubCliConnection) GetService(name string) (plugin_models.GetService_Model, error) {
	s.getServicesName = name

	return plugin_models.GetService_Model{
		Name: name,
		Guid: s.getServiceGuid,
	}, s.getServiceError
}

func (s *stubCliConnection) CliCommandWithoutTerminalOutput(args ...string) ([]string, error) {
	s.cliCommandWithoutTerminalOutputArgs = append(
		s.cliCommandWithoutTerminalOutputArgs,
		args,
	)

	output, ok := s.cliCommandWithoutTerminalOutputResponse[strings.Join(args, " ")]
	if !ok {
		output = "{}"
	}

	var err error
	switch args[0] {
	case "set-env":
		err = s.setEnvErrors[args[2]]
	}

	return strings.Split(output, "\n"), err
}

func (s *stubCliConnection) CliCommand(args ...string) ([]string, error) {
	var err error
	switch args[0] {
	case "create-user":
		err = s.createUserError
	case "create-user-provided-service":
		err = s.createServiceError
	case "bind-service":
		err = s.bindServiceError
	case "unbind-service":
		err = s.unbindServiceError
	case "delete-service":
		err = s.deleteServiceError
	case "push":
		err = s.pushAppError
	case "start":
		err = s.startAppError
	case "delete":
		err = s.deleteAppError
	}

	s.cliCommandArgs = append(s.cliCommandArgs, args)
	return nil, err
}

func (s *stubCliConnection) ApiEndpoint() (string, error) {
	return s.apiEndpoint, s.apiEndpointError
}

func (s *stubCliConnection) IsSSLDisabled() (bool, error) {
	return s.sslDisabled, nil
}

type stubLogger struct {
	fatalfMessage  string
	printfMessages []string
	printMessages  []string
}

func (l *stubLogger) Printf(format string, args ...interface{}) {
	l.printfMessages = append(l.printfMessages, fmt.Sprintf(format, args...))
}

func (l *stubLogger) Fatalf(format string, args ...interface{}) {
	l.fatalfMessage = fmt.Sprintf(format, args...)
	panic(l.fatalfMessage)
}

func (l *stubLogger) Print(a ...interface{}) {
	l.printMessages = append(l.printMessages, fmt.Sprint(a...))
}

type stubDownloader struct {
	path      string
	assetName string
}

func newStubDownloader() *stubDownloader {
	return &stubDownloader{}
}

func (s *stubDownloader) Download(assetName string) string {
	s.assetName = assetName
	return s.path
}
