package cloudcontroller_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCloudcontroller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cloudcontroller Suite")
}
