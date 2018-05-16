package helpers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"code.cloudfoundry.org/cf-drain-cli/acceptance"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var (
	logEmitterApp = "apps/ruby_simple"
	syslogDrain   = "apps/syslog-drain-listener"
)

func SilienceGinkgoWriter(f func()) {
	oldWriter := GinkgoWriter
	defer func() {
		GinkgoWriter = oldWriter
	}()
	GinkgoWriter = ioutil.Discard
	f()
}

func LogsTail(appName string) *Session {
	var s *Session
	SilienceGinkgoWriter(func() {
		s = cf.Cf("tail", appName, "--lines", "125")
	})

	return s
}

func LogsFollow(appName string) *Session {
	var s *Session
	SilienceGinkgoWriter(func() {
		s = cf.Cf("tail", "--follow", appName)
	})

	return s
}

func PushLogWriter() string {
	cfg := acceptance.Config()
	appName := generator.PrefixedRandomName("LOG-EMITTER", "")

	Eventually(cf.Cf(
		"push",
		appName,
		"-p", logEmitterApp,
	), cfg.AppPushTimeout).Should(Exit(0), "Failed to push app")

	return appName
}

func PushSyslogServer() string {
	cfg := acceptance.Config()
	appName := generator.PrefixedRandomName("SYSLOG-SERVER", "")

	Eventually(cf.Cf(
		"push",
		appName,
		"--health-check-type", "port",
		"-p", syslogDrain,
		"-b", "go_buildpack",
		"-f", syslogDrain+"/manifest.yml",
	), cfg.AppPushTimeout).Should(Exit(0), "Failed to push app")

	return appName
}

func WriteToLogsApp(doneChan chan struct{}, message, logWriterAppName string) {
	cfg := acceptance.Config()
	logUrl := fmt.Sprintf("https://%s.%s/log/%s", logWriterAppName, cfg.CFDomain, message)

	defer GinkgoRecover()
	for {
		select {
		case <-doneChan:
			return
		default:
			http.Get(logUrl)
			time.Sleep(3 * time.Second)
		}
	}
}

func SyslogDrainAddress(appName string) string {
	cfg := acceptance.Config()

	var address []byte
	Eventually(func() []byte {
		re, err := regexp.Compile("ADDRESS: \\|(.*)\\|")
		Expect(err).NotTo(HaveOccurred())

		logs := LogsTail(appName).Wait(cfg.DefaultTimeout)
		matched := re.FindSubmatch(logs.Out.Contents())
		if len(matched) < 2 {
			return nil
		}
		address = matched[1]
		return address
	}, cfg.DefaultTimeout).Should(Not(BeNil()))

	return string(address)
}
