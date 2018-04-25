package application_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/cf-drain-cli/internal/application"
)

var _ = Describe("BindDrainClient", func() {
	It("is a noop", func() {
		client := NewBindDrainClient()
		err := client.BindDrain("some-app-guid", "some-service-instance-guid")
		Expect(err).ToNot(HaveOccurred())
	})
})
