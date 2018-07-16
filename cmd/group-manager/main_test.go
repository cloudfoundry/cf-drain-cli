package main_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("GroupManager", func() {
	var (
		server         *httptest.Server
		reqs           chan *http.Request
		requestBodies  chan []byte
		responseBodies chan []byte
		cancel         func()
	)

	BeforeEach(func() {
		reqs = make(chan *http.Request, 100)
		requestBodies = make(chan []byte, 100)
		responseBodies = make(chan []byte, 100)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}

			reqs <- r
			requestBodies <- body

			if len(responseBodies) != 0 {
				w.Write(<-responseBodies)
			}
		}))
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	Context("when a source ID is given", func() {
		BeforeEach(func() {
			groupManagerEnv := []string{
				"SOURCE_ID=TEST_SOURCE",
				"UPDATE_INTERVAL=1ms",
				"GROUP_NAME=TEST_GROUP",
				`VCAP_APPLICATION={"cf_api":"https://api.test-server.com"}`,
				fmt.Sprintf("HTTP_PROXY=%s", server.URL),
			}

			path, err := gexec.Build("code.cloudfoundry.org/cf-drain-cli/cmd/group-manager")
			Expect(err).ToNot(HaveOccurred())

			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			cmd := exec.CommandContext(ctx, path)
			cmd.Env = groupManagerEnv
			err = cmd.Start()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			cancel()
		})

		It("adds the given source ID", func() {
			var req *http.Request
			Eventually(reqs).Should(Receive(&req))

			Expect(req.URL.Path).To(Equal("/v1/experimental/shard_group/TEST_GROUP"))
			Expect(req.URL.Host).To(Equal("log-cache.test-server.com"))
			Expect(req.URL.Scheme).To(Equal("http")) //we're communicating with a local proxy
			Expect(req.Method).To(Equal(http.MethodPut))

			var body []byte
			Eventually(requestBodies).Should(Receive(&body))

			Expect(body).To(MatchJSON(`{"sourceIds":["TEST_SOURCE"]}`))
		})
	})

	Context("when no source ID is given", func() {
		BeforeEach(func() {
			groupManagerEnv := []string{
				"UPDATE_INTERVAL=1ms",
				"GROUP_NAME=TEST_GROUP",
				`VCAP_APPLICATION={"cf_api":"https://api.test-server.com","space_id":"space-guid"}`,
				fmt.Sprintf("HTTP_PROXY=%s", server.URL),
			}

			path, err := gexec.Build("code.cloudfoundry.org/cf-drain-cli/cmd/group-manager")
			Expect(err).ToNot(HaveOccurred())

			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			cmd := exec.CommandContext(ctx, path)
			cmd.Env = groupManagerEnv
			err = cmd.Start()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			cancel()
		})

		It("gets source IDs from the CF API", func(done Done) {
			defer close(done)
			responseBodies <- []byte(`{
				"resources": [
					{"guid": "service-1"},
					{"guid": "service-2"},
					{"guid": "service-3"}
				]
			}`)

			var req *http.Request
			Eventually(reqs).Should(Receive(&req))

			// Throw away first request body because it is empty.
			<-requestBodies

			Expect(req.URL.Path).To(Equal("/v3/service_instances"))
			Expect(req.URL.Query().Get("space_guids")).To(Equal("space-guid"))
			Expect(req.URL.Host).To(Equal("api.test-server.com"))
			Expect(req.URL.Scheme).To(Equal("http")) //we're communicating with a local proxy
			Expect(req.Method).To(Equal(http.MethodGet))

			for i := 0; i < 3; i++ {
				Eventually(reqs).Should(Receive(&req))

				Expect(req.URL.Path).To(Equal("/v1/experimental/shard_group/TEST_GROUP"))
				Expect(req.URL.Host).To(Equal("log-cache.test-server.com"))
				Expect(req.URL.Scheme).To(Equal("http")) //we're communicating with a local proxy
				Expect(req.Method).To(Equal(http.MethodPut))

				var body []byte
				Eventually(requestBodies).Should(Receive(&body))

				Expect(body).To(MatchJSON(
					fmt.Sprintf(`{"sourceIds":["service-%d"]}`, i+1),
				))
			}
		}, 5)
	})
})
