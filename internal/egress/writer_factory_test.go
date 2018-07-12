package egress_test

import (
	"log"
	"net/url"

	"code.cloudfoundry.org/cf-drain-cli/internal/egress"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WriterFactory", func() {
	It("returns an https writer when the url begins with http", func() {
		url, err := url.Parse("https://the-syslog-endpoint.com")
		Expect(err).ToNot(HaveOccurred())

		writer := egress.NewWriter("source-host", url, egress.NetworkConfig{}, log.New(GinkgoWriter, "", 0))

		_, ok := writer.(*egress.HTTPSWriter)
		Expect(ok).To(BeTrue())
	})

	It("returns a tcp writer when the url begins with syslog://", func() {
		url, err := url.Parse("syslog://the-syslog-endpoint.com")
		Expect(err).ToNot(HaveOccurred())

		writer := egress.NewWriter("source-host", url, egress.NetworkConfig{}, log.New(GinkgoWriter, "", 0))

		_, ok := writer.(*egress.TCPWriter)
		Expect(ok).To(BeTrue())
	})

	It("returns a syslog-tls writer when the url begins with syslog-tls://", func() {
		url, err := url.Parse("syslog-tls://the-syslog-endpoint.com")
		Expect(err).ToNot(HaveOccurred())

		writer := egress.NewWriter("source-host", url, egress.NetworkConfig{}, log.New(GinkgoWriter, "", 0))

		_, ok := writer.(*egress.TLSWriter)
		Expect(ok).To(BeTrue())
	})
})
