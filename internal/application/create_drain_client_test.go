package application_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/application"
)

var _ = Describe("CreateDrainClient", func() {
	var c *application.CreateDrainClient

	BeforeEach(func() {
		c = application.NewCreateDrainClient()
	})

	It("pushes a syslog_forwarder app", func() {
		err := c.CreateDrain("some-name", "some-url", "some-space", "all")
		Expect(err).ToNot(HaveOccurred())
	})

	It("fatally logs for unknown drain types", func() {
		invalidType := fmt.Sprintf("invalid-%d", time.Now().UnixNano())
		err := c.CreateDrain("some-name", "some-url", "some-space", invalidType)
		Expect(err).To(MatchError(fmt.Sprintf("invalid drain type: %s", invalidType)))
	})
})
