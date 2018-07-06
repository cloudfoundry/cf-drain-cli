package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/cf-drain-cli/acceptance/helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"time"
	"code.cloudfoundry.org/cf-drain-cli/acceptance"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("StructuredLog", func() {
	var appName string
	BeforeEach(func() {
		appName = PushLogWriter()
	})

	AfterEach(func() {
		deleteApp := func(appName string) {
			CF("delete", appName, "-f", "-r")
		}

		deleteApp(appName)
	})

	It("creates a structured log drain and deletes it", func() {
		CF(
			"enable-structured-logging",
			appName,
			"dogstatsd",
			"--drain-name", "drain-name",
		)

		Eventually(func() string {
			s := cf.Cf("drains")
			Eventually(s, acceptance.Config().DefaultTimeout).Should(gexec.Exit(0))
			return string(append(s.Out.Contents(), s.Err.Contents()...))
		}, acceptance.Config().DefaultTimeout+3*time.Minute).Should(And(
			ContainSubstring("drain-name"),
			ContainSubstring("prism://dogstatsd"),
		))
	})

})