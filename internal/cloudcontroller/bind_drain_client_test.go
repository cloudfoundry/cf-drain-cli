package cloudcontroller_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var _ = Describe("BindDrainClient", func() {
	var (
		curler *stubCurler
		c      *cloudcontroller.BindDrainClient
	)

	BeforeEach(func() {
		curler = newStubCurler()
		c = cloudcontroller.NewBindDrainClient(curler)
	})

	It("POSTs the correct body", func() {
		err := c.BindDrain("some-app-guid", "some-drain-guid")
		Expect(err).ToNot(HaveOccurred())

		Expect(curler.methods).To(ConsistOf("POST"))
		Expect(curler.URLs).To(ConsistOf("/v2/service_bindings"))
		Expect(curler.bodies).To(ConsistOf(MatchJSON(`
        {
          "service_instance_guid": "some-drain-guid",
          "app_guid": "some-app-guid"
        }`,
		)))
	})

	It("returns an error if the POST fails", func() {
		curler.errs["/v2/service_bindings"] = errors.New("some-error")
		err := c.BindDrain("some-app-guid", "some-drain-guid")
		Expect(err).To(MatchError("some-error"))
	})
})
