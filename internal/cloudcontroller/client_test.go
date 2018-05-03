package cloudcontroller_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var _ = Describe("CloudControllerClient", func() {
	var (
		c      *cloudcontroller.Client
		curler *stubCurler
	)

	It("retrieves EnvVars", func() {
		forwarderGUID := "5b40bdd6-4587-43aa-b5a5-d1d410560c03"

		curler = newStubCurler()
		curler.resps[fmt.Sprintf("/v3/apps/%s/env", forwarderGUID)] = `{
			"environment_variables": {
				 "DRAIN_URL": "syslog://my-syslog-drain.com",
				 "DRAIN_TYPE": "all"
			 }
		 }`
		c = cloudcontroller.NewClient(curler)

		envs, err := c.EnvVars(forwarderGUID)

		Expect(err).ToNot(HaveOccurred())
		Expect(curler.URLs).To(ConsistOf(
			fmt.Sprintf("/v3/apps/%s/env", forwarderGUID),
		))
		Expect(envs["DRAIN_TYPE"]).To(Equal("all"))
		Expect(envs["DRAIN_URL"]).To(Equal("syslog://my-syslog-drain.com"))
	})

	It("returns an error if it fails to fetch app env vars from cloudcontroller", func() {
		curlErr := errors.New("some-err")
		curler = newStubCurler()
		curler.errs[fmt.Sprintf("/v3/apps/%s/env", "bad-guid")] = curlErr

		c = cloudcontroller.NewClient(curler)
		_, err := c.EnvVars("bad-guid")

		Expect(err).To(MatchError(
			errors.New("failed to fetch app environment variables: some-err"),
		))
	})
})
