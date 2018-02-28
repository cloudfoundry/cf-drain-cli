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

	getAppName  string
	getAppError error

	getServicesName  string
	getServicesError error
	getServicesApps  []string

	cliCommandWithoutTerminalOutputArgs     [][]string
	cliCommandWithoutTerminalOutputResponse map[string]string

	cliCommandArgs     [][]string
	createServiceError error
	bindServiceError   error
	unbindServiceError error
	deleteServiceError error
	pushAppError       error

	currentSpaceGuid  string
	currentSpaceError error

	apiEndpoint    string
	apiEndpointErr error

	setEnvErrors map[string]error
}

func newStubCliConnection() *stubCliConnection {
	return &stubCliConnection{
		cliCommandWithoutTerminalOutputResponse: make(map[string]string),
		setEnvErrors:                            make(map[string]error),
	}
}

func (s *stubCliConnection) GetApp(name string) (plugin_models.GetAppModel, error) {
	s.getAppName = name
	return plugin_models.GetAppModel{}, s.getAppError
}

func (s *stubCliConnection) GetCurrentSpace() (plugin_models.Space, error) {
	return plugin_models.Space{
		plugin_models.SpaceFields{
			Guid: s.currentSpaceGuid,
		},
	}, s.currentSpaceError
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
	}

	s.cliCommandArgs = append(s.cliCommandArgs, args)
	return nil, err
}

func (s *stubCliConnection) ApiEndpoint() (string, error) {
	return s.apiEndpoint, s.apiEndpointErr
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
