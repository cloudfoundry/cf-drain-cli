package helpers

import (
	"time"

	"code.cloudfoundry.org/cf-drain-cli/acceptance"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func CF(args ...string) {
	defer GinkgoRecover()
	EventuallyWithOffset(1, cf.Cf(args...), acceptance.Config().DefaultTimeout).Should(Exit(0))
}

func CFWithTimeout(timeout time.Duration, args ...string) {
	defer GinkgoRecover()
	EventuallyWithOffset(1, cf.Cf(args...), timeout).Should(Exit(0))
}
