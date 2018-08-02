package main_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/rfc5424"
	"github.com/golang/protobuf/jsonpb"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var logTimestamp = int64(1257894000000000000)

var _ = Describe("Main", func() {
	var (
		proxy *httptest.Server

		capiReqs     chan *http.Request
		capiRespCode int
		capiResps    chan []byte

		rlpReqs chan *http.Request
		rlpResp chan []byte

		fakeSyslog   *httptest.Server
		syslogReqs   chan *http.Request
		syslogBodies chan []byte

		cancel func()
	)

	BeforeEach(func() {
		capiRespCode = http.StatusOK
		capiResps = make(chan []byte, 100)
		capiReqs = make(chan *http.Request)

		rlpResp = make(chan []byte, 100)
		rlpReqs = make(chan *http.Request)
		proxy = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v2/read":
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				rlpReqs <- r

				flusher := w.(http.Flusher)
				flusher.Flush()

				for data := range rlpResp {
					w.Write(data)
					flusher.Flush()
				}
			case "/v3/service_instances":
				capiReqs <- r

				if capiRespCode != 200 {
					w.WriteHeader(capiRespCode)
				}
				w.Write(<-capiResps)
			default:
				panic(fmt.Sprintf("unhandled request: %s", r.URL.Path))
			}
		}))

		syslogReqs = make(chan *http.Request, 100)
		syslogBodies = make(chan []byte, 100)
		fakeSyslog = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			Expect(err).ToNot(HaveOccurred())
			defer r.Body.Close()

			syslogReqs <- r
			syslogBodies <- body
		}))
	})

	AfterEach(func() {
		cancel()
		close(rlpResp)
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	Context("single source id", func() {
		BeforeEach(func() {
			forwarderEnv := []string{
				"SOURCE_ID=service-1",
				"SOURCE_HOSTNAME=TEST_HOSTNAME",
				`VCAP_APPLICATION={"application_id":"forwarder-id","cf_api":"https://api.test-server.com", "space_id": "space-guid"}`,
				"SKIP_CERT_VERIFY=true",
				fmt.Sprintf("HTTP_PROXY=%s", proxy.URL),
				fmt.Sprintf("SYSLOG_URL=%s", fakeSyslog.URL),
			}

			path, err := gexec.Build("code.cloudfoundry.org/cf-drain-cli/cmd/syslog-forwarder")
			Expect(err).ToNot(HaveOccurred())

			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			cmd := exec.CommandContext(ctx, path)
			cmd.Env = forwarderEnv
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			err = cmd.Start()
			Expect(err).ToNot(HaveOccurred())
		})

		It("forwards the logs from the RLP to the syslog endpoint", func() {
			rlpResp <- []byte(buildSSEMessage("service-1"))

			var rlpReq *http.Request
			Eventually(rlpReqs).Should(Receive(&rlpReq))
			Expect(rlpReq.URL.Path).To(Equal("/v2/read"))
			Expect(rlpReq.URL.Host).To(Equal("log-stream.test-server.com"))
			Expect(rlpReq.Method).To(Equal(http.MethodGet))
			Expect(rlpReq.URL.Query()).To(HaveKeyWithValue("source_id", []string{"service-1"}))
			Expect(rlpReq.URL.Query()).To(HaveKeyWithValue("shard_id", []string{"forwarder-id"}))
			Expect(rlpReq.URL.Query()).To(HaveKey("log"))
			Expect(rlpReq.URL.Query()).To(HaveKey("counter"))
			Expect(rlpReq.URL.Query()).To(HaveKey("gauge"))

			var syslogReq *http.Request
			Eventually(syslogReqs).Should(Receive(&syslogReq))
			Expect(syslogReq.Method).To(Equal(http.MethodPost))

			expected := messageBytes("service-1")

			var actual []byte
			Eventually(syslogBodies).Should(Receive(&actual))

			Expect(expected).To(Equal(actual))
		})
	})

	Context("whole space", func() {
		BeforeEach(func() {
			forwarderEnv := []string{
				"UPDATE_INTERVAL=500ms",
				"SOURCE_HOSTNAME=TEST_HOSTNAME",
				"SKIP_CERT_VERIFY=true",
				`VCAP_APPLICATION={"application_id":"forwarder-id","cf_api":"https://api.test-server.com", "space_id": "space-guid"}`,
				fmt.Sprintf("HTTP_PROXY=%s", proxy.URL),
				fmt.Sprintf("SYSLOG_URL=%s", fakeSyslog.URL),
			}

			path, err := gexec.Build("code.cloudfoundry.org/cf-drain-cli/cmd/syslog-forwarder")
			Expect(err).ToNot(HaveOccurred())

			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			cmd := exec.CommandContext(ctx, path)
			cmd.Env = forwarderEnv
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			err = cmd.Start()
			Expect(err).ToNot(HaveOccurred())
		})

		It("forwards the logs for service instances in a space from the RLP to the syslog endpoint", func() {
			capiResps <- []byte(serviceInstancesBody)
			rlpResp <- []byte(buildSSEMessage("service-1"))
			rlpResp <- []byte(buildSSEMessage("service-2"))
			rlpResp <- []byte(buildSSEMessage("service-3"))

			var capiReq *http.Request
			Eventually(capiReqs).Should(Receive(&capiReq))
			Expect(capiReq.URL.Path).To(Equal("/v3/service_instances"))
			Expect(capiReq.URL.Query()).To(HaveKeyWithValue("space_guids", []string{"space-guid"}))

			var sourceIDs []string
			var rlpReq *http.Request
			Eventually(rlpReqs).Should(Receive(&rlpReq))
			Expect(rlpReq.URL.Path).To(Equal("/v2/read"))
			Expect(rlpReq.URL.Host).To(Equal("log-stream.test-server.com"))
			Expect(rlpReq.Method).To(Equal(http.MethodGet))
			Expect(rlpReq.URL.Query()).To(HaveKey("log"))
			Expect(rlpReq.URL.Query()).To(HaveKey("counter"))
			Expect(rlpReq.URL.Query()).To(HaveKey("gauge"))
			Expect(rlpReq.URL.Query()).To(HaveKeyWithValue("shard_id", []string{"forwarder-id"}))
			sourceIDs = append(sourceIDs, rlpReq.URL.Query()["source_id"]...)

			Eventually(rlpReqs).Should(Receive(&rlpReq))
			Expect(rlpReq.URL.Query()).To(HaveKey("log"))
			Expect(rlpReq.URL.Query()).To(HaveKey("counter"))
			Expect(rlpReq.URL.Query()).To(HaveKey("gauge"))
			Expect(rlpReq.URL.Query()).To(HaveKeyWithValue("shard_id", []string{"forwarder-id"}))
			sourceIDs = append(sourceIDs, rlpReq.URL.Query()["source_id"]...)

			Eventually(rlpReqs).Should(Receive(&rlpReq))
			Expect(rlpReq.URL.Query()).To(HaveKey("log"))
			Expect(rlpReq.URL.Query()).To(HaveKey("counter"))
			Expect(rlpReq.URL.Query()).To(HaveKey("gauge"))
			Expect(rlpReq.URL.Query()).To(HaveKeyWithValue("shard_id", []string{"forwarder-id"}))
			sourceIDs = append(sourceIDs, rlpReq.URL.Query()["source_id"]...)

			Expect(sourceIDs).To(ConsistOf(
				"service-1",
				"service-2",
				"service-3",
			))

			Eventually(syslogReqs).Should(HaveLen(3))

			var bodies [][]byte
			for len(syslogReqs) > 0 {
				req := <-syslogReqs
				Expect(req.Method).To(Equal(http.MethodPost))
				bodies = append(bodies, <-syslogBodies)
			}

			Expect(bodies).To(ConsistOf(
				messageBytes("service-1"),
				messageBytes("service-2"),
				messageBytes("service-3"),
			))

			//the forwarder adapts to changes in the space
			capiResps <- []byte(serviceInstancesBody2)
			rlpResp <- []byte(buildSSEMessage("service-4"))

			Eventually(capiReqs, 30).Should(Receive(&capiReq))
			Expect(capiReq.URL.Path).To(Equal("/v3/service_instances"))
			Expect(capiReq.URL.Query()).To(HaveKeyWithValue("space_guids", []string{"space-guid"}))

			Eventually(rlpReqs).Should(Receive(&rlpReq))
			Expect(rlpReq.URL.Path).To(Equal("/v2/read"))
			Expect(rlpReq.URL.Host).To(Equal("log-stream.test-server.com"))
			Expect(rlpReq.Method).To(Equal(http.MethodGet))
			Expect(rlpReq.URL.Query()).To(HaveKeyWithValue("source_id", []string{"service-4"}))
			Expect(rlpReq.URL.Query()).To(HaveKey("log"))
			Expect(rlpReq.URL.Query()).To(HaveKey("counter"))
			Expect(rlpReq.URL.Query()).To(HaveKey("gauge"))

			var actual []byte
			Eventually(syslogBodies).Should(Receive(&actual))

			Expect(messageBytes("service-4")).To(Equal(actual))
		})
	})
})

func messageBytes(appName string) []byte {
	msg := &rfc5424.Message{
		Timestamp: time.Unix(0, logTimestamp).UTC(),
		AppName:   appName,
		Hostname:  "TEST_HOSTNAME." + appName,
		Priority:  rfc5424.Priority(14),
		ProcessID: "[APP/PROC/WEB/0]",
		Message:   []byte("log body\n"),
	}

	b, err := msg.MarshalBinary()
	Expect(err).ToNot(HaveOccurred())

	return b
}

func buildSSEMessage(sourceID string) string {
	m := jsonpb.Marshaler{}
	s, err := m.MarshalToString(&loggregator_v2.EnvelopeBatch{
		Batch: []*loggregator_v2.Envelope{
			{
				SourceId:   sourceID,
				InstanceId: "0",
				Timestamp:  logTimestamp,
				Tags: map[string]string{
					"source_type": "APP/PROC/WEB",
				},
				Message: &loggregator_v2.Envelope_Log{
					Log: &loggregator_v2.Log{
						Payload: []byte("log body"),
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("data: %s\n\n", s)
}

var serviceInstancesBody = `
{
	"resources": [
		{
			"guid": "service-1"
		},
		{
			"guid": "service-2"
		},
		{
			"guid": "service-3"
		}
	]
}
`
var serviceInstancesBody2 = `
{
	"resources": [
		{
			"guid": "service-4"
		}
	]
}
`
