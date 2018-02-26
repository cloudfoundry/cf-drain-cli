package cloudcontroller_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var _ = Describe("CreateDrainClient", func() {
	var (
		curler *stubCurler
		c      *cloudcontroller.CreateDrainClient
	)

	BeforeEach(func() {
		curler = newStubCurler()
		c = cloudcontroller.NewCreateDrainClient(curler)
	})

	It("POSTs the request to the Curler", func() {
		err := c.CreateDrain("some-name", "some-url", "some-space")
		Expect(err).ToNot(HaveOccurred())
		Expect(curler.methods).To(ConsistOf("POST"))
		Expect(curler.URLs).To(ConsistOf("/v2/user_provided_service_instances"))
		Expect(curler.bodies).To(ConsistOf(MatchJSON(`
		{
		   "space_guid": "some-space",
		   "name": "some-name",
		   "syslog_drain_url": "some-url"
		}`,
		)))
	})

	It("returns an error if the POST fails", func() {
		curler.errs["/v2/user_provided_service_instances"] = errors.New("some-error")
		err := c.CreateDrain("some-name", "some-url", "some-space")
		Expect(err).To(MatchError("some-error"))
	})
})
