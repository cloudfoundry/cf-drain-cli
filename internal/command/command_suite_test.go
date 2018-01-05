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
	cliCommandWithoutTerminalOutputError    error
	cliCommandWithoutTerminalOutputResponse map[string]string

	cliCommandArgs     [][]string
	createServiceError error
	bindServiceError   error
	unbindServiceError error
	deleteServiceError error

	currentSpaceGuid  string
	currentSpaceError error
}

func newStubCliConnection() *stubCliConnection {
	return &stubCliConnection{
		cliCommandWithoutTerminalOutputResponse: make(map[string]string),
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

	return strings.Split(output, "\n"), s.cliCommandWithoutTerminalOutputError
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
	}

	s.cliCommandArgs = append(s.cliCommandArgs, args)
	return nil, err
}

type stubLogger struct {
	fatalfMessage  string
	printfMessages []string
}

func (l *stubLogger) Printf(format string, args ...interface{}) {
	l.printfMessages = append(l.printfMessages, fmt.Sprintf(format, args...))
}

func (l *stubLogger) Fatalf(format string, args ...interface{}) {
	l.fatalfMessage = fmt.Sprintf(format, args...)
	panic(l.fatalfMessage)
}
