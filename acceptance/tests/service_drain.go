package tests

import (
	"fmt"
	"path"
	"regexp"
	"strings"
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
		drainsRegex       = `App\s+Drain\s+Type\s+URL
LOG-EMITTER-1--[0-9a-f]{16}\s+cf-drain-[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}\s+Logs\s+syslog://\d+[.]\d+[.]\d+[.]\d+:\d+`
	)

	BeforeEach(func() {
		interrupt = make(chan struct{}, 1)

		var wg sync.WaitGroup
		defer wg.Wait()

		wg.Add(3)
		go func() {
			defer wg.Done()
			defer GinkgoRecover()
			listenerAppName = PushSyslogServer()
		}()
		go func() {
			defer wg.Done()
			defer GinkgoRecover()
			logWriterAppName1 = PushLogWriter()
		}()
		go func() {
			defer wg.Done()
			defer GinkgoRecover()
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
			defer GinkgoRecover()
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

		Eventually(logs, acceptance.Config().DefaultTimeout+3*time.Minute).Should(Say(randomMessage1))
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

		Eventually(logs, acceptance.Config().DefaultTimeout+3*time.Minute).Should(Say(randomMessage1))
		Eventually(logs, acceptance.Config().DefaultTimeout+3*time.Minute).Should(Say(randomMessage2))
	})

	It("drains all apps in space to a syslog endpoint", func() {
		syslogDrainURL := "syslog://" + SyslogDrainAddress(listenerAppName)
		drainName := fmt.Sprintf("some-drain-%d", time.Now().UnixNano())

		execPath, err := Build("code.cloudfoundry.org/cf-drain-cli/space_drain")
		Expect(err).ToNot(HaveOccurred())

		defer CleanupBuildArtifacts()

		CFWithTimeout(
			1*time.Minute,
			"drain-space",
			syslogDrainURL,
			"--drain-name", drainName,
			"--path", path.Dir(execPath),
			"--force",
		)

		defer CF("delete", drainName, "-f", "-r")

		randomMessage1 := generator.PrefixedRandomName("RANDOM-MESSAGE-A", "LOG")
		randomMessage2 := generator.PrefixedRandomName("RANDOM-MESSAGE-B", "LOG")

		logs = LogsFollow(listenerAppName)

		go WriteToLogsApp(interrupt, randomMessage1, logWriterAppName1)
		go WriteToLogsApp(interrupt, randomMessage2, logWriterAppName2)

		Eventually(logs, acceptance.Config().DefaultTimeout+3*time.Minute).Should(Say(randomMessage1))
		Eventually(logs, acceptance.Config().DefaultTimeout+3*time.Minute).Should(Say(randomMessage2))

		// Apps are the first column listed.
		re := regexp.MustCompile(fmt.Sprintf(`^(%s)`, drainName))

		Consistently(func() string {
			s := cf.Cf("drains")
			Eventually(s, acceptance.Config().DefaultTimeout).Should(Exit(0))

			for _, line := range strings.Split(string(s.Out.Contents()), "\n") {
				if re.Match([]byte(line)) {
					return line
				}
			}

			return ""
		}, acceptance.Config().DefaultTimeout).ShouldNot(ContainSubstring(drainName))
	})

	It("deletes space-drain but not other drains", func() {
		syslogDrainURL := "syslog://" + SyslogDrainAddress(listenerAppName)
		drainName := fmt.Sprintf("some-drain-%d", time.Now().UnixNano())
		singleDrainName := fmt.Sprintf("single-some-drain-%d", time.Now().UnixNano())

		execPath, err := Build("code.cloudfoundry.org/cf-drain-cli/space_drain")
		Expect(err).ToNot(HaveOccurred())

		defer CleanupBuildArtifacts()

		CFWithTimeout(
			1*time.Minute,
			"drain-space",
			syslogDrainURL,
			"--drain-name", drainName,
			"--path", path.Dir(execPath),
			"--force",
		)

		CF(
			"drain",
			logWriterAppName1,
			syslogDrainURL,
			"--drain-name", singleDrainName,
		)

		Eventually(func() string {
			s := cf.Cf("drains")
			Eventually(s, acceptance.Config().DefaultTimeout).Should(Exit(0))
			return string(append(s.Out.Contents(), s.Err.Contents()...))
		}, acceptance.Config().DefaultTimeout+3*time.Minute).Should(And(
			ContainSubstring(drainName),
			ContainSubstring(singleDrainName),
		))

		CFWithTimeout(
			1*time.Minute,
			"delete-drain-space",
			drainName,
			"--force",
		)

		Eventually(func() string {
			s := cf.Cf("drains")
			Eventually(s, acceptance.Config().DefaultTimeout).Should(Exit(0))
			return string(append(s.Out.Contents(), s.Err.Contents()...))
		}, acceptance.Config().DefaultTimeout+3*time.Minute).ShouldNot(ContainSubstring(drainName))

		Consistently(func() string {
			s := cf.Cf("drains")
			Eventually(s, acceptance.Config().DefaultTimeout).Should(Exit(0))
			return string(append(s.Out.Contents(), s.Err.Contents()...))
		}, acceptance.Config().DefaultTimeout).Should(ContainSubstring(singleDrainName))
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

	It("drain-space reports error when space-drain with same drain-name exists", func() {
		syslogDrainURL := "syslog://" + SyslogDrainAddress(listenerAppName)

		execPath, err := Build("code.cloudfoundry.org/cf-drain-cli/space_drain")
		Expect(err).ToNot(HaveOccurred())

		defer CleanupBuildArtifacts()

		CFWithTimeout(
			1*time.Minute,
			"drain-space",
			syslogDrainURL,
			"--drain-name", "some-space-drain",
			"--path", path.Dir(execPath),
			"--force",
		)

		drainSpace := cf.Cf(
			"drain-space",
			syslogDrainURL,
			"--drain-name", "some-space-drain",
			"--path", path.Dir(execPath),
			"--force",
		)

		Eventually(drainSpace).Should(Say("A drain with that name already exists. Use --drain-name to create a drain with a different name."))
	})

	It("a space-drain cannot drain to itself or to any other space-drains", func() {
		syslogDrainURL1 := "syslog://space-drain-1.papertrail.com"
		syslogDrainURL2 := "syslog://space-drain-2.splunk.com"

		execPath, err := Build("code.cloudfoundry.org/cf-drain-cli/space_drain")
		Expect(err).ToNot(HaveOccurred())

		defer CleanupBuildArtifacts()

		CFWithTimeout(
			1*time.Minute,
			"drain-space",
			syslogDrainURL1,
			"--drain-name", "space-drain-papertrail",
			"--path", path.Dir(execPath),
			"--force",
		)

		CFWithTimeout(
			1*time.Minute,
			"drain-space",
			syslogDrainURL2,
			"--drain-name", "space-drain-splunk",
			"--path", path.Dir(execPath),
			"--force",
		)

		papertrailDrainRegex := `(?m:^space-drain-papertrail)`

		Eventually(func() string {
			s := cf.Cf("drains")
			Eventually(s, acceptance.Config().DefaultTimeout).Should(Exit(0))
			return string(append(s.Out.Contents(), s.Err.Contents()...))
		}, acceptance.Config().DefaultTimeout).ShouldNot(MatchRegexp(papertrailDrainRegex))
	})
})
