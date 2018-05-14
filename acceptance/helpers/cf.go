package helpers

import (
	"code.cloudfoundry.org/cf-drain-cli/acceptance"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func CF(args ...string) {
	Eventually(cf.Cf(args...), acceptance.Config().DefaultTimeout).Should(Exit(0))
}
