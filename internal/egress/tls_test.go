package egress_test

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"time"

	"code.cloudfoundry.org/cf-drain-cli/internal/egress"
	"code.cloudfoundry.org/cf-drain-cli/internal/testhelper"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TLSWriter", func() {
	var (
		tlsConfig *tls.Config

		certFile = testhelper.Cert("syslog.crt")
		keyFile  = testhelper.Cert("syslog.key")
		env      = buildLogEnvelope("APP", "2", "just a test", loggregator_v2.Log_OUT)
		netConf  = egress.NetworkConfig{
			WriteTimeout:   time.Second,
			SkipCertVerify: true,
		}
	)

	BeforeEach(func() {
		tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
		Expect(err).ToNot(HaveOccurred())

		tlsConfig = &tls.Config{
			Certificates:       []tls.Certificate{tlsCert},
			InsecureSkipVerify: true,
		}
	})

	It("speaks TLS", func() {
		listener, err := tls.Listen("tcp", ":0", tlsConfig)
		Expect(err).ToNot(HaveOccurred())

		url, _ := url.Parse(fmt.Sprintf("syslog-tls://%s", listener.Addr()))
		binding := &egress.URLBinding{
			Hostname: "test-hostname",
			URL:      url,
		}
		writer := egress.NewTLSWriter(
			binding,
			netConf,
		)
		defer writer.Close()

		var conn net.Conn
		go func() {
			var err error
			conn, err = listener.Accept()
			Expect(err).ToNot(HaveOccurred())

			// Note: for some odd reason you have to do a read off of the TLS
			// connection before the dial will succeed. We should probably
			// investigate.
			empty := make([]byte, 0)
			conn.Read(empty)
		}()

		Expect(writer.Write(env)).To(Succeed())

		buf := bufio.NewReader(conn)

		actual, err := buf.ReadString('\n')
		Expect(err).ToNot(HaveOccurred())

		expected := fmt.Sprintf("101 <14>1 1970-01-01T00:00:00.012345+00:00 test-hostname.test-app-id test-app-id [APP/2] - - just a test\n")
		Expect(actual).To(Equal(expected))
	})
})
