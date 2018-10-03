package egress_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"

	"code.cloudfoundry.org/cf-drain-cli/internal/egress"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/rfc5424"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTPWriter", func() {
	var netConf egress.NetworkConfig

	BeforeEach(func() {
		netConf = egress.NetworkConfig{
			SkipCertVerify: true,
		}
	})

	It("errors when ssl validation is enabled", func() {
		drain := newMockOKDrain()
		b := buildURLBinding(drain.URL, "test-hostname")
		netConf.SkipCertVerify = false

		writer := egress.NewHTTPSWriter(
			b,
			netConf,
		)

		env := buildLogEnvelope("APP", "1", "just a test", loggregator_v2.Log_OUT)
		Expect(writer.Write(env)).ToNot(Succeed())
	})

	It("errors on an invalid syslog message", func() {
		drain := newMockOKDrain()

		b := buildURLBinding(
			drain.URL,
			"test-hostname",
		)
		writer := egress.NewHTTPSWriter(
			b,
			netConf,
		)

		env := buildLogEnvelope("APP", "1", "just a test", loggregator_v2.Log_OUT)
		env.SourceId = "test-app-id-012345678901234567890012345678901234567890"

		Expect(writer.Write(env)).ToNot(Succeed())
	})

	It("errors when the http POST fails", func() {
		drain := newMockErrorDrain()

		b := buildURLBinding(
			drain.URL,
			"test-hostname",
		)

		writer := egress.NewHTTPSWriter(
			b,
			netConf,
		)

		env := buildLogEnvelope("APP", "1", "just a test", loggregator_v2.Log_OUT)
		Expect(writer.Write(env)).ToNot(Succeed())
	})

	It("writes syslog formatted messages to http drain", func() {
		drain := newMockOKDrain()

		b := buildURLBinding(
			drain.URL,
			"test-hostname",
		)

		writer := egress.NewHTTPSWriter(
			b,
			netConf,
		)

		env1 := buildLogEnvelope("APP", "1", "just a test", loggregator_v2.Log_OUT)
		Expect(writer.Write(env1)).To(Succeed())
		env2 := buildLogEnvelope("CELL", "5", "log from cell", loggregator_v2.Log_ERR)
		Expect(writer.Write(env2)).To(Succeed())
		env3 := buildLogEnvelope("CELL", "", "log from cell", loggregator_v2.Log_ERR)
		Expect(writer.Write(env3)).To(Succeed())

		Expect(drain.messages).To(HaveLen(3))
		expected := &rfc5424.Message{
			AppName:   "test-app-id",
			Hostname:  "test-hostname.test-app-id",
			Priority:  rfc5424.Priority(14),
			ProcessID: "[APP/1]",
			Message:   []byte("just a test\n"),
		}
		Expect(drain.messages[0].AppName).To(Equal(expected.AppName))
		Expect(drain.messages[0].Hostname).To(Equal(expected.Hostname))
		Expect(drain.messages[0].Priority).To(BeEquivalentTo(expected.Priority))
		Expect(drain.messages[0].ProcessID).To(Equal(expected.ProcessID))
		Expect(drain.messages[0].Message).To(Equal(expected.Message))

		expected = &rfc5424.Message{
			AppName:   "test-app-id",
			Hostname:  "test-hostname.test-app-id",
			Priority:  rfc5424.Priority(11),
			ProcessID: "[CELL/5]",
			Message:   []byte("log from cell\n"),
		}
		Expect(drain.messages[1].AppName).To(Equal(expected.AppName))
		Expect(drain.messages[1].Hostname).To(Equal(expected.Hostname))
		Expect(drain.messages[1].Priority).To(BeEquivalentTo(expected.Priority))
		Expect(drain.messages[1].ProcessID).To(Equal(expected.ProcessID))
		Expect(drain.messages[1].Message).To(Equal(expected.Message))

		Expect(drain.messages[2].ProcessID).To(Equal("[CELL]"))
	})

	It("writes gauge metrics to the http drain", func() {
		drain := newMockOKDrain()

		b := buildURLBinding(
			drain.URL,
			"test-hostname",
		)

		writer := egress.NewHTTPSWriter(
			b,
			netConf,
		)

		env1 := buildGaugeEnvelope("1")
		Expect(writer.Write(env1)).To(Succeed())

		Expect(drain.messages).To(HaveLen(5))

		Expect(drain.messages[0].StructuredData).To(HaveLen(1))
		Expect(drain.messages[0].StructuredData[0].ID).To(Equal("gauge@47450"))

		sdValues := func(msgs []*rfc5424.Message, name string) []string {
			var sd rfc5424.StructuredData
			for _, msg := range msgs {
				if msg.StructuredData[0].Parameters[0].Value == name {
					sd = msg.StructuredData[0]
					break
				}
			}

			data := make([]string, 0, 3)
			for _, param := range sd.Parameters {
				data = append(data, param.Value)
			}

			return data
		}

		Expect(sdValues(drain.messages, "cpu")).To(ConsistOf("cpu", "0.23", "percentage"))
		Expect(sdValues(drain.messages, "disk")).To(ConsistOf("disk", "1234", "bytes"))
		Expect(sdValues(drain.messages, "disk_quota")).To(ConsistOf("disk_quota", "1024", "bytes"))
		Expect(sdValues(drain.messages, "memory")).To(ConsistOf("memory", "5423", "bytes"))
		Expect(sdValues(drain.messages, "memory_quota")).To(ConsistOf("memory_quota", "8000", "bytes"))
	})

	It("writes counter metrics to the http drain", func() {
		drain := newMockOKDrain()

		b := buildURLBinding(
			drain.URL,
			"test-hostname",
		)

		writer := egress.NewHTTPSWriter(
			b,
			netConf,
		)

		env1 := buildCounterEnvelope("1")
		Expect(writer.Write(env1)).To(Succeed())

		Expect(drain.messages).To(HaveLen(1))

		Expect(drain.messages[0].StructuredData).To(HaveLen(1))
		Expect(drain.messages[0].StructuredData[0].ID).To(Equal("counter@47450"))

		Expect(drain.messages[0].StructuredData[0].Parameters[0].Name).To(Equal("name"))
		Expect(drain.messages[0].StructuredData[0].Parameters[0].Value).To(Equal("some-counter"))

		Expect(drain.messages[0].StructuredData[0].Parameters[1].Name).To(Equal("total"))
		Expect(drain.messages[0].StructuredData[0].Parameters[1].Value).To(Equal("99"))

		Expect(drain.messages[0].StructuredData[0].Parameters[2].Name).To(Equal("delta"))
		Expect(drain.messages[0].StructuredData[0].Parameters[2].Value).To(Equal("1"))
	})

	It("emits an syslog metric for each message", func() {
		drain := newMockOKDrain()

		b := buildURLBinding(
			drain.URL,
			"test-hostname",
		)

		writer := egress.NewHTTPSWriter(
			b,
			netConf,
		)

		env := buildLogEnvelope("APP", "1", "just a test", loggregator_v2.Log_OUT)
		writer.Write(env)
	})

	It("ignores non-log envelopes", func() {
		drain := newMockOKDrain()

		b := buildURLBinding(
			drain.URL,
			"test-hostname",
		)

		writer := egress.NewHTTPSWriter(
			b,
			netConf,
		)

		counterEnv := buildTimerEnvelope()
		logEnv := buildLogEnvelope("APP", "2", "just a test", loggregator_v2.Log_OUT)

		Expect(writer.Write(counterEnv)).To(Succeed())
		Expect(writer.Write(logEnv)).To(Succeed())
	})
})

type SpyDrain struct {
	*httptest.Server
	messages []*rfc5424.Message
}

func newMockOKDrain() *SpyDrain {
	return newMockDrain(http.StatusOK)
}

func newMockErrorDrain() *SpyDrain {
	return newMockDrain(http.StatusBadRequest)
}

func newMockDrain(status int) *SpyDrain {
	drain := &SpyDrain{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		message := &rfc5424.Message{}

		body, err := ioutil.ReadAll(r.Body)
		Expect(err).ToNot(HaveOccurred())
		defer r.Body.Close()

		err = message.UnmarshalBinary(body)
		Expect(err).ToNot(HaveOccurred())

		drain.messages = append(drain.messages, message)
		w.WriteHeader(status)
	})
	server := httptest.NewTLSServer(handler)
	drain.Server = server
	return drain
}

func buildURLBinding(u, hostname string) *egress.URLBinding {
	parsedURL, _ := url.Parse(u)

	return &egress.URLBinding{
		URL:      parsedURL,
		Hostname: hostname,
	}
}
