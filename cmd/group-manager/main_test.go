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
		server *httptest.Server
		reqs   chan *http.Request
		bodies chan []byte
		cancel func()
	)

	BeforeEach(func() {
		reqs = make(chan *http.Request)
		bodies = make(chan []byte)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}

			reqs <- r
			bodies <- body
		}))

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

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	It("adds the given source ID", func() {
		var req *http.Request
		Eventually(reqs).Should(Receive(&req))

		Expect(req.URL.Path).To(Equal("/v1/experimental/shard_group/TEST_GROUP"))
		Expect(req.URL.Host).To(Equal("log-cache.test-server.com"))
		Expect(req.URL.Scheme).To(Equal("http")) //we're communicating with a local proxy
		Expect(req.Method).To(Equal(http.MethodPut))

		var body []byte
		Eventually(bodies).Should(Receive(&body))

		Expect(body).To(MatchJSON(`{"sourceIds":["TEST_SOURCE"]}`))
	})
})
