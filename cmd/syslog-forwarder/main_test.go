package main_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"code.cloudfoundry.org/rfc5424"
)

var _ = Describe("Main", func() {
	var (
		fakeLogCache *httptest.Server
		logCacheReqs chan *http.Request
		logCacheResp []byte

		fakeSyslog   *httptest.Server
		syslogReqs   chan *http.Request
		syslogBodies chan *rfc5424.Message

		cancel func()
	)

	BeforeEach(func() {
		logCacheReqs = make(chan *http.Request)
		fakeLogCache = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logCacheReqs <- r
			w.Write(logCacheResp)
		}))

		syslogReqs = make(chan *http.Request)
		syslogBodies = make(chan *rfc5424.Message)
		fakeSyslog = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			message := &rfc5424.Message{}

			body, err := ioutil.ReadAll(r.Body)
			Expect(err).ToNot(HaveOccurred())
			defer r.Body.Close()

			err = message.UnmarshalBinary(body)
			Expect(err).ToNot(HaveOccurred())

			syslogReqs <- r
			syslogBodies <- message
		}))

		forwarderEnv := []string{
			"GROUP_NAME=TEST_GROUP",
			"SOURCE_HOSTNAME=TEST_HOSTNAME",
			`VCAP_APPLICATION={"cf_api":"https://api.test-server.com"}`,
			"SKIP_CERT_VERIFY=true",
			fmt.Sprintf("HTTP_PROXY=%s", fakeLogCache.URL),
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

	AfterEach(func() {
		cancel()
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	It("forwards the logs from the LogCacheGroup to the syslog endpoint", func() {
		logCacheResp = []byte(envelope)

		var lcReq *http.Request
		Eventually(logCacheReqs).Should(Receive(&lcReq))
		Expect(lcReq.URL.Path).To(Equal("/v1/experimental/shard_group/TEST_GROUP"))
		Expect(lcReq.Method).To(Equal(http.MethodGet))

		var syslogReq *http.Request
		Eventually(syslogReqs).Should(Receive(&syslogReq))
		Expect(syslogReq.Method).To(Equal(http.MethodPost))

		expected := &rfc5424.Message{
			AppName:   "app-name",
			Hostname:  "TEST_HOSTNAME", //What is the hostname?
			Priority:  rfc5424.Priority(14),
			ProcessID: "[APP/PROC/WEB/0]",
			Message:   []byte("log body\n"),
		}

		Eventually(syslogBodies).Should(Receive(&expected))
	})
})

var envelope = `{
					"envelopes": {
						"batch": [
							{
								"source_id": "app-name",
								"timestamp":"1257894000000000000",
								"instance_id":"0",
								"tags":{
									"source_type":"APP/PROC/WEB"
								},
								"log":{
									"payload":"bG9nIGJvZHk="
								}
							}
						]
					}
				}`
