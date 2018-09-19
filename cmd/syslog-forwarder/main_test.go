package main_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
		serviceResps chan []byte
		appResps     chan []byte

		rlpReqs chan *http.Request
		rlpResp map[string]chan []byte

		fakeSyslog   *httptest.Server
		syslogReqs   chan *http.Request
		syslogBodies chan []byte

		cancel func()
		cmd    *exec.Cmd
	)

	BeforeEach(func() {
		capiRespCode = http.StatusOK
		serviceResps = make(chan []byte, 100)
		appResps = make(chan []byte, 100)
		capiReqs = make(chan *http.Request, 100)

		rlpResp = make(map[string]chan []byte)
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

				sourceID := r.URL.Query().Get("source_id")
				c := rlpResp[sourceID]

				for data := range c {
					w.Write(data)
					flusher.Flush()
				}
			case "/v3/service_instances":
				capiReqs <- r

				if capiRespCode != 200 {
					w.WriteHeader(capiRespCode)
				}

				w.Write(<-serviceResps)
			case "/v3/apps":
				capiReqs <- r

				if capiRespCode != 200 {
					w.WriteHeader(capiRespCode)
				}

				w.Write(<-appResps)
			default:
				w.WriteHeader(http.StatusNotFound)
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
		cmd.Wait()

		for _, c := range rlpResp {
			close(c)
		}
		close(syslogReqs)

		proxy.CloseClientConnections()
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	Context("single source id", func() {
		BeforeEach(func() {
			forwarderEnv := []string{
				"INCLUDE_SERVICES=true",
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
			cmd = exec.CommandContext(ctx, path)
			cmd.Env = forwarderEnv
			cmd.Stderr = GinkgoWriter
			cmd.Stdout = GinkgoWriter
			err = cmd.Start()
			Expect(err).ToNot(HaveOccurred())
		})

		It("forwards the logs from the RLP to the syslog endpoint", func() {
			serviceResps <- []byte(serviceInstancesBody)
			rlpResp["service-1"] = make(chan []byte, 100)
			rlpResp["service-1"] <- []byte(buildSSEMessage("service-1"))

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

			expected := messageBytes("service-1-name", "service-1")

			var actual []byte
			Eventually(syslogBodies).Should(Receive(&actual))
			Expect(expected).To(Equal(string(actual)))
		})
	})

	Context("whole space", func() {
		BeforeEach(func() {
			forwarderEnv := []string{
				"INCLUDE_SERVICES=true",
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
			cmd = exec.CommandContext(ctx, path)
			cmd.Env = forwarderEnv
			cmd.Stderr = GinkgoWriter
			cmd.Stdout = GinkgoWriter
			err = cmd.Start()
			Expect(err).ToNot(HaveOccurred())
		}, 5)

		It("forwards the logs for services and non-forwarder app instances in a space from the RLP to the syslog endpoint", func() {
			serviceResps <- []byte(serviceInstancesBody)
			appResps <- []byte(appsBody)

			rlpResp["service-1"] = make(chan []byte, 100)
			rlpResp["service-1"] <- []byte(buildSSEMessage("service-1"))
			rlpResp["service-2"] = make(chan []byte, 100)
			rlpResp["service-2"] <- []byte(buildSSEMessage("service-2"))
			rlpResp["service-3"] = make(chan []byte, 100)
			rlpResp["service-3"] <- []byte(buildSSEMessage("service-3"))
			rlpResp["app-1"] = make(chan []byte, 100)
			rlpResp["app-1"] <- []byte(buildSSEMessage("app-1"))
			rlpResp["app-2"] = make(chan []byte, 100)
			rlpResp["app-2"] <- []byte(buildSSEMessage("app-2"))
			rlpResp["app-3"] = make(chan []byte, 100)
			rlpResp["app-3"] <- []byte(buildSSEMessage("app-3"))

			numSources := 6

			var capiReq *http.Request
			Eventually(capiReqs).Should(Receive(&capiReq))
			Expect(capiReq.URL.Path).To(Equal("/v3/service_instances"))
			Expect(capiReq.URL.Query()).To(HaveKeyWithValue("space_guids", []string{"space-guid"}))

			Eventually(capiReqs).Should(Receive(&capiReq))
			Expect(capiReq.URL.Path).To(Equal("/v3/apps"))
			Expect(capiReq.URL.Query()).To(HaveKeyWithValue("space_guids", []string{"space-guid"}))

			var sourceIDs []string
			var rlpReq *http.Request
			for i := 0; i < numSources; i++ {
				Eventually(rlpReqs).Should(Receive(&rlpReq))
				Expect(rlpReq.URL.Path).To(Equal("/v2/read"))
				Expect(rlpReq.URL.Host).To(Equal("log-stream.test-server.com"))
				Expect(rlpReq.Method).To(Equal(http.MethodGet))
				Expect(rlpReq.URL.Query()).To(HaveKey("log"))
				Expect(rlpReq.URL.Query()).To(HaveKey("counter"))
				Expect(rlpReq.URL.Query()).To(HaveKey("gauge"))
				Expect(rlpReq.URL.Query()).To(HaveKeyWithValue("shard_id", []string{"forwarder-id"}))
				sourceIDs = append(sourceIDs, rlpReq.URL.Query()["source_id"]...)
			}

			Consistently(rlpReqs).ShouldNot(Receive(&rlpReq))

			Expect(sourceIDs).To(ConsistOf(
				"service-1",
				"service-2",
				"service-3",
				"app-1",
				"app-2",
				"app-3",
			))

			Eventually(syslogReqs).Should(HaveLen(numSources))

			var bodies []string
			for len(syslogReqs) > 0 {
				req := <-syslogReqs
				Expect(req.Method).To(Equal(http.MethodPost))
				bodies = append(bodies, string(<-syslogBodies))
			}

			Expect(bodies).To(ConsistOf(
				messageBytes("service-1-name", "service-1"),
				messageBytes("service-2-name", "service-2"),
				messageBytes("service-3-name", "service-3"),
				messageBytes("app-1-name", "app-1"),
				messageBytes("app-2-name", "app-2"),
				messageBytes("app-3-name", "app-3"),
			))

			//the forwarder adapts to changes in the space
			serviceResps <- []byte(serviceInstancesBody2)
			appResps <- []byte(emptyJSON)

			rlpResp["service-4"] = make(chan []byte, 100)
			rlpResp["service-4"] <- []byte(buildSSEMessage("service-4"))

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

			Expect(messageBytes("service-4-name", "service-4")).To(Equal(string(actual)))
		}, 5)
	})

	Context("only apps", func() {
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
			cmd = exec.CommandContext(ctx, path)
			cmd.Env = forwarderEnv
			cmd.Stderr = GinkgoWriter
			cmd.Stdout = GinkgoWriter
			err = cmd.Start()
			Expect(err).ToNot(HaveOccurred())
		})

		It("forwards the logs for apps in a space from the RLP to the syslog endpoint", func() {
			serviceResps <- []byte(serviceInstancesBody)
			appResps <- []byte(appsBody)

			rlpResp["app-1"] = make(chan []byte, 100)
			rlpResp["app-1"] <- []byte(buildSSEMessage("app-1"))
			rlpResp["app-2"] = make(chan []byte, 100)
			rlpResp["app-2"] <- []byte(buildSSEMessage("app-2"))
			rlpResp["app-3"] = make(chan []byte, 100)
			rlpResp["app-3"] <- []byte(buildSSEMessage("app-3"))

			numLogs := 3

			var capiReq *http.Request
			Eventually(capiReqs).Should(Receive(&capiReq))
			Expect(capiReq.URL.Path).ToNot(Equal("/v3/service_instances"))
			Expect(capiReq.URL.Path).To(Equal("/v3/apps"))
			Expect(capiReq.URL.Query()).To(HaveKeyWithValue("space_guids", []string{"space-guid"}))

			var sourceIDs []string
			var rlpReq *http.Request
			for i := 0; i < numLogs; i++ {
				Eventually(rlpReqs).Should(Receive(&rlpReq))
				Expect(rlpReq.URL.Path).To(Equal("/v2/read"))
				Expect(rlpReq.URL.Host).To(Equal("log-stream.test-server.com"))
				Expect(rlpReq.Method).To(Equal(http.MethodGet))
				Expect(rlpReq.URL.Query()).To(HaveKey("log"))
				Expect(rlpReq.URL.Query()).To(HaveKey("counter"))
				Expect(rlpReq.URL.Query()).To(HaveKey("gauge"))
				Expect(rlpReq.URL.Query()).To(HaveKeyWithValue("shard_id", []string{"forwarder-id"}))
				sourceIDs = append(sourceIDs, rlpReq.URL.Query()["source_id"]...)
			}

			Consistently(rlpReqs).ShouldNot(Receive(&rlpReq))

			Expect(sourceIDs).To(ConsistOf(
				"app-1",
				"app-2",
				"app-3",
			))

			Eventually(syslogReqs, 10).Should(HaveLen(numLogs))

			var bodies []string
			for len(syslogReqs) > 0 {
				req := <-syslogReqs
				Expect(req.Method).To(Equal(http.MethodPost))
				bodies = append(bodies, string(<-syslogBodies))
			}

			Expect(bodies).To(ConsistOf(
				messageBytes("app-1-name", "app-1"),
				messageBytes("app-2-name", "app-2"),
				messageBytes("app-3-name", "app-3"),
			))
		})
	})
})

func messageBytes(hostnameSuffix, appID string) string {
	msg := &rfc5424.Message{
		Timestamp: time.Unix(0, logTimestamp).UTC(),
		AppName:   appID,
		Hostname:  "TEST_HOSTNAME." + hostnameSuffix,
		Priority:  rfc5424.Priority(14),
		ProcessID: "[APP/PROC/WEB/0]",
		Message:   []byte("log body\n"),
	}

	b, err := msg.MarshalBinary()
	Expect(err).ToNot(HaveOccurred())

	return string(b)
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
			"guid": "service-1",
			"name": "service-1-name"
		},
		{
			"guid": "service-2",
			"name": "service-2-name"

		},
		{
			"guid": "service-3",
			"name": "service-3-name"
		}
	]
}
`
var serviceInstancesBody2 = `
{
	"resources": [
		{
			"guid": "service-4",
			"name": "service-4-name"
		}
	]
}
`
var appsBody = `
{
	"resources": [
		{
			"guid": "app-1",
			"name": "app-1-name"
		},
		{
			"guid": "app-2",
			"name": "app-2-name"
		},
		{
			"guid": "app-3",
			"name": "app-3-name"
		},
		{
			"guid": "forwarder-id",
			"name": "syslog-forwarder"
		}
	]
}
`

var emptyJSON = `{
	"resources": []
}`
