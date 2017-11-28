package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCfSyslogCli(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CfSyslogCli Suite")
}
