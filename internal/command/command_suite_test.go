package command_test

import (
	"errors"
	"fmt"
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

	getServiceName  string
	getServiceError error

	cliCommandArgs     [][]string
	createServiceError error
	bindServiceError   error
}

func newStubCliConnection() *stubCliConnection {
	return &stubCliConnection{
		getServiceError: errors.New("no-such-service"),
	}
}

func (s *stubCliConnection) GetApp(name string) (plugin_models.GetAppModel, error) {
	s.getAppName = name
	return plugin_models.GetAppModel{}, s.getAppError
}

func (s *stubCliConnection) GetService(name string) (plugin_models.GetService_Model, error) {
	s.getServiceName = name
	return plugin_models.GetService_Model{}, s.getServiceError
}

func (s *stubCliConnection) CliCommand(args ...string) ([]string, error) {
	var err error
	switch args[0] {
	case "create-user-provided-service":
		err = s.createServiceError
	case "bind-service":
		err = s.bindServiceError
	}

	s.cliCommandArgs = append(s.cliCommandArgs, args)
	return nil, err
}

type stubLogger struct {
	fatalfMessage string
}

func (l *stubLogger) Fatalf(format string, args ...interface{}) {
	l.fatalfMessage = fmt.Sprintf(format, args...)
	panic(l.fatalfMessage)
}
