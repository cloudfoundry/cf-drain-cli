package drain

import (
	"fmt"
	"sync"
	"time"

	"code.cloudfoundry.org/cf-drain-cli/acceptance"
	. "code.cloudfoundry.org/cf-drain-cli/acceptance/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
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
		drains            *Session
		drainsRegex       = `App                              Drain                                          Type      URL                        AdapterType
LOG-EMITTER-1--[0-9a-f]{16}  cf-drain-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}  Logs      syslog://\d+[.]\d+[.]\d+[.]\d+:\d+  service`
	)

	BeforeEach(func() {
		interrupt = make(chan struct{}, 1)

		var wg sync.WaitGroup
		defer wg.Wait()

		wg.Add(3)
		go func() {
			defer wg.Done()
			listenerAppName = PushSyslogServer()
		}()
		go func() {
			defer wg.Done()
			logWriterAppName1 = PushLogWriter()
		}()
		go func() {
			defer wg.Done()
			logWriterAppName2 = PushLogWriter()
		}()
	})

	AfterEach(func() {
		if logs != nil {
			logs.Kill()
		}
		if drains != nil {
			drains.Kill()
		}

		close(interrupt)

		var wg sync.WaitGroup
		defer wg.Wait()
		deleteApp := func(appName string) {
			defer wg.Done()
			CF("delete", appName, "-f", "-r")
		}

		wg.Add(3)
		go deleteApp(logWriterAppName1)
		go deleteApp(logWriterAppName2)
		go deleteApp(listenerAppName)
	})

	It("drains an app's logs to syslog endpoint", func() {
		syslogDrainURL := "syslog://" + SyslogDrainAddress(listenerAppName)

		CF(
			"drain",
			logWriterAppName1,
			syslogDrainURL,
		)

		randomMessage1 := generator.PrefixedRandomName("RANDOM-MESSAGE-A", "LOG")
		randomMessage2 := generator.PrefixedRandomName("RANDOM-MESSAGE-B", "LOG")

		logs = LogsFollow(listenerAppName)

		go WriteToLogsApp(interrupt, randomMessage1, logWriterAppName1)
		go WriteToLogsApp(interrupt, randomMessage2, logWriterAppName2)

		Eventually(logs, acceptance.Config().DefaultTimeout+1*time.Minute).Should(Say(randomMessage1))
		Consistently(logs, 10).ShouldNot(Say(randomMessage2))
	})

	It("binds an app to a syslog endpoint", func() {
		syslogDrainURL := "syslog://" + SyslogDrainAddress(listenerAppName)
		drainName := fmt.Sprintf("some-drain-%d", time.Now().UnixNano())

		CF(
			"drain",
			logWriterAppName1,
			syslogDrainURL,
			"--drain-name", drainName,
		)

		CF(
			"bind-drain",
			logWriterAppName2,
			drainName,
		)

		randomMessage1 := generator.PrefixedRandomName("RANDOM-MESSAGE-A", "LOG")
		randomMessage2 := generator.PrefixedRandomName("RANDOM-MESSAGE-B", "LOG")

		logs = LogsFollow(listenerAppName)

		go WriteToLogsApp(interrupt, randomMessage1, logWriterAppName1)
		go WriteToLogsApp(interrupt, randomMessage2, logWriterAppName2)

		Eventually(logs, acceptance.Config().DefaultTimeout+1*time.Minute).Should(Say(randomMessage1))
		Eventually(logs, acceptance.Config().DefaultTimeout+1*time.Minute).Should(Say(randomMessage2))
	})

	It("drains all apps in space to a syslog endpoint", func() {
		syslogDrainURL := "syslog://" + SyslogDrainAddress(listenerAppName)
		drainName := fmt.Sprintf("some-drain-%d", time.Now().UnixNano())

		CFWithTimeout(
			1*time.Minute,
			"drain-space",
			"--drain-url", syslogDrainURL,
			"--drain-name", drainName,
			"--username", acceptance.Config().CFAdminUser,
			"--password", acceptance.Config().CFAdminPassword,
			"--force",
		)

		defer CF("delete", "space-drain", "-f", "-r")

		randomMessage1 := generator.PrefixedRandomName("RANDOM-MESSAGE-A", "LOG")
		randomMessage2 := generator.PrefixedRandomName("RANDOM-MESSAGE-B", "LOG")

		logs = LogsFollow(listenerAppName)

		go WriteToLogsApp(interrupt, randomMessage1, logWriterAppName1)
		go WriteToLogsApp(interrupt, randomMessage2, logWriterAppName2)

		Eventually(logs, acceptance.Config().DefaultTimeout+1*time.Minute).Should(Say(randomMessage1))
		Eventually(logs, acceptance.Config().DefaultTimeout+1*time.Minute).Should(Say(randomMessage2))
	})

	It("lists the all the drains", func() {
		syslogDrainURL := "syslog://" + SyslogDrainAddress(listenerAppName)

		CF(
			"drain",
			logWriterAppName1,
			syslogDrainURL,
		)

		drains = cf.Cf("drains")
		Eventually(drains).Should(Say(drainsRegex))
	})

	It("deletes the drain", func() {
		syslogDrainURL := "syslog://" + SyslogDrainAddress(listenerAppName)

		CF(
			"drain",
			logWriterAppName1,
			syslogDrainURL,
			"--drain-name",
			"cf-drain-8075db89-6080-4b14-93e5-10d69f24d7e1",
		)

		drains = cf.Cf("drains")
		Eventually(drains).Should(Say(drainsRegex))

		CF(
			"delete-drain",
			"cf-drain-8075db89-6080-4b14-93e5-10d69f24d7e1",
			"--force", // Skip confirmation
		)

		drains = cf.Cf("drains")
		Consistently(drains, 10).ShouldNot(Say("cf-drain-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}"))
	})
})
