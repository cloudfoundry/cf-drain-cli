package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSyslogForwarder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SyslogForwarder Suite")
}
