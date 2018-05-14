package drain

import (
	"time"

	"code.cloudfoundry.org/cf-drain-cli/acceptance"
	. "code.cloudfoundry.org/cf-drain-cli/acceptance/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("ServiceDrain", func() {

	var (
		listenerAppName   string
		logWriterAppName1 string
		logWriterAppName2 string
		interrupt         chan struct{}
		logs              *Session
	)

	BeforeEach(func() {
		interrupt = make(chan struct{}, 1)

		listenerAppName = PushSyslogServer()
		logWriterAppName1 = PushLogWriter()
		logWriterAppName2 = PushLogWriter()
	})

	AfterEach(func() {
		logs.Kill()
		close(interrupt)

		CF("delete", logWriterAppName1, "-f", "-r")
		CF("delete", logWriterAppName2, "-f", "-r")
		CF("delete", listenerAppName, "-f", "-r")
	})

	It("drains an app's logs to syslog endpoint", func() {
		syslogDrainURL := "syslog://" + SyslogDrainAddress(listenerAppName)

		CF("drain",
			logWriterAppName1,
			syslogDrainURL,
			"--adapter-type",
			"service",
		)

		randomMessage1 := generator.PrefixedRandomName("RANDOM-MESSAGE-A", "LOG")
		randomMessage2 := generator.PrefixedRandomName("RANDOM-MESSAGE-B", "LOG")

		logs = LogsFollow(listenerAppName)

		go WriteToLogsApp(interrupt, randomMessage1, logWriterAppName1)
		go WriteToLogsApp(interrupt, randomMessage2, logWriterAppName2)

		Eventually(logs, acceptance.Config().DefaultTimeout+3*time.Minute).Should(Say(randomMessage1))
		Consistently(logs, 10).ShouldNot(Say(randomMessage2))
	})
})
