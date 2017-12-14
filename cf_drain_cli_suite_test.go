package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCfDrainCli(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CfDrainCli Suite")
}
