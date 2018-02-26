package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSpaceManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SpaceManager Suite")
}
