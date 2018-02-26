package cloudcontroller_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var _ = Describe("AppListerClient", func() {
	var (
		curler *stubCurler
		c      *cloudcontroller.AppListerClient
	)

	BeforeEach(func() {
		curler = newStubCurler()
		c = cloudcontroller.NewAppListerClient(curler)
	})

	It("requests all apps in the space", func() {
		curler.resps["/v2/apps?q=space_guid:some-space"] = `
		{
			"resources": [
			{
				"metadata":{"guid":"a"}
			},
			{
				"metadata":{"guid":"b"}
			}
			]
		}
		`
		apps, err := c.ListApps("some-space")
		Expect(err).ToNot(HaveOccurred())
		Expect(curler.methods).To(ConsistOf("GET"))
		Expect(curler.URLs).To(ConsistOf("/v2/apps?q=space_guid:some-space"))
		Expect(apps).To(ConsistOf("a", "b"))
	})

	It("returns an error if the GET fails", func() {
		curler.errs["/v2/apps?q=space_guid:some-space"] = errors.New("some-error")
		_, err := c.ListApps("some-space")
		Expect(err).To(MatchError("some-error"))
	})

	It("returns an error if the JSON is invalid", func() {
		curler.resps["/v2/apps?q=space_guid:some-space"] = `invalid`
		_, err := c.ListApps("some-space")
		Expect(err).To(HaveOccurred())
	})
})
