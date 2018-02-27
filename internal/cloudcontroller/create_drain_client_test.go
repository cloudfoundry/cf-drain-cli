package cloudcontroller_test

import (
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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
		err := c.CreateDrain("some-name", "some-url", "some-space", "all")
		Expect(err).ToNot(HaveOccurred())
		Expect(curler.methods).To(ConsistOf("POST"))
		Expect(curler.URLs).To(ConsistOf("/v2/user_provided_service_instances"))
		Expect(curler.bodies).To(ConsistOf(MatchJSON(`
		{
		   "space_guid": "some-space",
		   "name": "some-name",
		   "syslog_drain_url": "some-url?drain-type=all"
		}`,
		)))
	})

	It("returns an error if the POST fails", func() {
		curler.errs["/v2/user_provided_service_instances"] = errors.New("some-error")
		err := c.CreateDrain("some-name", "some-url", "some-space", "all")
		Expect(err).To(MatchError("some-error"))
	})

	DescribeTable("drain types", func(drainType string) {
		err := c.CreateDrain("some-name", "some-url", "some-space", drainType)

		Expect(err).ToNot(HaveOccurred())
		Expect(curler.bodies).To(ConsistOf(MatchJSON(fmt.Sprintf(`
			{
			   "space_guid": "some-space",
			   "name": "some-name",
			   "syslog_drain_url": "some-url?drain-type=%s"
			}`, drainType),
		)))
	},
		Entry("drain-type=all", "all"),
		Entry("drain-type=metrics", "metrics"),
		Entry("drain-type=logs", "logs"),
	)

	It("fatally logs for unknown drain types", func() {
		invalidType := fmt.Sprintf("invalid-%d", time.Now().UnixNano())
		err := c.CreateDrain("some-name", "some-url", "some-space", invalidType)
		Expect(err).To(MatchError(fmt.Sprintf("invalid drain type: %s", invalidType)))
		Expect(curler.URLs).To(BeEmpty())
	})
})
