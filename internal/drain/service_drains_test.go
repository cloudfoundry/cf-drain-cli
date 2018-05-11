package drain_test

import (
	"errors"
	"strings"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
	"code.cloudfoundry.org/cf-drain-cli/internal/drain"
	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServiceDrainLister", func() {
	var (
		cli         *stubCliConnection
		curler      *stubCurler
		appLister   *spyAppLister
		envProvider *spyEnvProvider
		c           *drain.ServiceDrainLister
		key         string
	)

	BeforeEach(func() {
		cli = newStubCliConnection()
		curler = newStubCurler()
		appLister = newSpyAppLister()
		envProvider = newSpyEnvProvider()

		c = drain.NewServiceDrainLister(cli, curler, appLister, envProvider)
	})

	It("only displays syslog services", func() {
		var noDrainServiceInstancesJSON = `{
		   "total_results": 1,
		   "total_pages": 1,
		   "prev_url": null,
		   "next_url": null,
		   "resources": [
		      {
		         "entity": {
		            "name": "other-service-1",
		            "syslog_drain_url": "",
		            "service_bindings_url": "/v2/user_provided_service_instances/other-service-1/service_bindings"
		         }
		      }
		   ]
		}`
		key = "/v2/user_provided_service_instances?q=space_guid:space-guid"
		curler.resps[key] = noDrainServiceInstancesJSON
		d, err := c.Drains("space-guid")

		Expect(err).ToNot(HaveOccurred())
		Expect(d).To(HaveLen(0))
	})

	Context("when requesting service instances succeeds", func() {
		BeforeEach(func() {
			key = "/v2/user_provided_service_instances?q=space_guid:space-guid"
			curler.resps[key] = serviceInstancesJSONpage1
			key = "/v2/user_provided_service_instances?q=space_guid:space-guid&page:2"
			curler.resps[key] = serviceInstancesJSONpage2
		})

		Context("when requesting service bindings succeeds", func() {
			BeforeEach(func() {
				key = "/v2/user_provided_service_instances/drain-1/service_bindings"
				curler.resps[key] = serviceBindingsJSON1page1
				key = "/v2/user_provided_service_instances/drain-1/service_bindings&page:2"
				curler.resps[key] = serviceBindingsJSON1page2
				key = "/v2/user_provided_service_instances/drain-2/service_bindings"
				curler.resps[key] = serviceBindingsJSON2
			})

			Context("when requesting app names succeeds", func() {
				BeforeEach(func() {
					key = "/v3/apps?guids=app-1,app-2,app-1"
					curler.resps[key] = appJSONpage1
					key = "/v3/apps?guids=app-1,app-2,app-1&page=2"
					curler.resps[key] = appJSONpage2
				})

				It("returns every drain", func() {
					d, err := c.Drains("space-guid")
					Expect(err).ToNot(HaveOccurred())
					Expect(d).To(HaveLen(2))

					Expect(d[0].Name).To(Equal("drain-1"))
					Expect(d[0].Guid).To(Equal("guid-1"))
					Expect(d[0].Apps).To(Equal([]string{"My App One", "My App Two"}))
					Expect(d[0].AppGuids).To(Equal([]string{"app-1", "app-2"}))
					Expect(d[0].Type).To(Equal("logs"))
					Expect(d[0].DrainURL).To(Equal("syslog://your-app.cf-app.com"))
					Expect(d[0].AdapterType).To(Equal("service"))

					Expect(d[1].Name).To(Equal("drain-2"))
					Expect(d[1].Guid).To(Equal("guid-2"))
					Expect(d[1].Apps).To(Equal([]string{"My App One"}))
					Expect(d[1].AppGuids).To(Equal([]string{"app-1"}))
					Expect(d[1].Type).To(Equal("metrics"))
					Expect(d[1].DrainURL).To(Equal("https://your-app2.cf-app.com?drain-type=metrics"))
					Expect(d[1].AdapterType).To(Equal("service"))

					// 7 => 2 service fetch + (2 app fetches) +  (3 app name fetches)
					Expect(curler.methods).To(ConsistOf("GET", "GET", "GET", "GET", "GET", "GET", "GET"))
					Expect(curler.bodies).To(ConsistOf("", "", "", "", "", "", ""))
				})

			})

			It("returns the error if requesting the apps fails", func() {
				key = "/v3/apps?guids=app-1,app-2,app-1"
				curler.errs[key] = errors.New("some error")

				_, err := c.Drains("space-guid")
				Expect(err).To(MatchError("some error"))
			})

			It("returns the error if unmarshalling the apps fails", func() {
				key = "/v3/apps?guids=app-1,app-2,app-1"
				curler.resps[key] = "not a json"

				_, err := c.Drains("space-guid")
				Expect(err).To(HaveOccurred())
			})
		})

		It("returns the error if requesting the service bindings fails", func() {
			key = "/v2/user_provided_service_instances/drain-1/service_bindings"
			curler.errs[key] = errors.New("some error")

			_, err := c.Drains("space-guid")
			Expect(err).To(MatchError("some error"))
		})

		It("returns the error if unmarshalling the service bindings fails", func() {
			key = "/v2/user_provided_service_instances/drain-1/service_bindings"
			curler.resps[key] = "no json"

			_, err := c.Drains("space-guid")
			Expect(err).To(HaveOccurred())
		})
	})

	It("returns the error if requesting the service instances fails", func() {
		key = "/v2/user_provided_service_instances?q=space_guid:space-guid"
		curler.errs[key] = errors.New("some error")

		_, err := c.Drains("space-guid")
		Expect(err).To(MatchError("some error"))
	})

	It("returns the error if unmarshalling the service instances response fails", func() {
		key = "/v2/user_provided_service_instances?q=space_guid:space-guid"
		curler.resps[key] = "not a JSON"

		_, err := c.Drains("space-guid")
		Expect(err).To(HaveOccurred())
	})

	Describe("TypeFromDrainURL", func() {
		It("returns default type logs if no query parameters", func() {
			drainType, _ := c.TypeFromDrainURL("https://papertrail.com")
			Expect(drainType).To(Equal("logs"))
		})

		It("returns logs type if drain-type query parameter is logs", func() {
			drainType, _ := c.TypeFromDrainURL("https://papertrail.com?drain-type=logs")
			Expect(drainType).To(Equal("logs"))
		})

		It("returns metrics type if drain-type query parameter is metrics", func() {
			drainType, _ := c.TypeFromDrainURL("https://papertrail.com?drain-type=metrics")
			Expect(drainType).To(Equal("metrics"))
		})

		It("returns all type if drain-type query parameter is all", func() {
			drainType, _ := c.TypeFromDrainURL("https://papertrail.com?drain-type=all")
			Expect(drainType).To(Equal("all"))
		})

		It("returns default type if url is invalid", func() {
			drainType, _ := c.TypeFromDrainURL("!!!so invalid")
			Expect(drainType).To(Equal("logs"))
		})
	})

	Describe("DeleteDrainAndUser", func() {

		It("unbinds and deletes the service and deletes drain", func() {
			key = "/v2/user_provided_service_instances?q=space_guid:space-guid"
			curler.resps[key] = serviceInstancesJSONpage1
			key = "/v2/user_provided_service_instances?q=space_guid:space-guid&page:2"
			curler.resps[key] = serviceInstancesJSONpage2

			key = "/v2/user_provided_service_instances/drain-1/service_bindings"
			curler.resps[key] = serviceBindingsJSON1page1
			key = "/v2/user_provided_service_instances/drain-1/service_bindings&page:2"
			curler.resps[key] = serviceBindingsJSON1page2
			key = "/v2/user_provided_service_instances/drain-2/service_bindings"
			curler.resps[key] = serviceBindingsJSON2

			key = "/v3/apps?guids=app-1,app-2,app-1"
			curler.resps[key] = appJSONpage1
			key = "/v3/apps?guids=app-1,app-2,app-1&page=2"
			curler.resps[key] = appJSONpage2

			appLister.apps = []cloudcontroller.App{
				cloudcontroller.App{
					Name: "drain-1",
					Guid: "00000000-0000-0000-0000-000000000000",
				},
				cloudcontroller.App{
					Name: "app-1",
					Guid: "22222222-2222-2222-2222-222222222222",
				},
				cloudcontroller.App{
					Name: "app-2",
					Guid: "33333333-3333-3333-3333-333333333333",
				},
			}

			envProvider.envs = map[string]map[string]string{
				"00000000-0000-0000-0000-000000000000": {
					"DRAIN_SCOPE": "single",
					"SOURCE_ID":   "22222222-2222-2222-2222-222222222222",
					"DRAIN_TYPE":  "logs",
					"SYSLOG_URL":  "syslog://the-syslog-drain.com",
				},
				"11111111-1111-1111-1111-111111111111": {
					"DRAIN_SCOPE": "space",
					"DRAIN_TYPE":  "all",
					"DRAIN_URL":   "https://the-syslog-drain.com",
				},
			}

			cli.getServicesName = "drain-1"
			cli.getServicesApps = []string{"app-1", "app-2"}

			ok, err := c.DeleteDrainAndUser("space-guid", "drain-1")

			Expect(ok).To(BeTrue())
			Expect(err).ShouldNot(HaveOccurred())

			// unbind and delete service instance
			Expect(cli.cliCommandArgs).To(HaveLen(3))
			Expect(cli.cliCommandArgs[0]).To(Equal([]string{
				"unbind-service", "app-1", "drain-1",
			}))
			Expect(cli.cliCommandArgs[1]).To(Equal([]string{
				"unbind-service", "app-2", "drain-1",
			}))
			Expect(cli.cliCommandArgs[2]).To(Equal([]string{
				"delete-service", "drain-1", "-f",
			}))
		})

		It("deletes space drain app if scope is space", func() {
			appLister.apps = []cloudcontroller.App{
				cloudcontroller.App{
					Name: "space-drain-1",
					Guid: "space-guid",
				},
				cloudcontroller.App{
					Name: "app-1",
					Guid: "22222222-2222-2222-2222-222222222222",
				},
				cloudcontroller.App{
					Name: "app-2",
					Guid: "33333333-3333-3333-3333-333333333333",
				},
			}

			envProvider.envs = map[string]map[string]string{
				"space-guid": {
					"DRAIN_SCOPE": "space",
					"DRAIN_TYPE":  "logs",
					"SYSLOG_URL":  "syslog://the-syslog-drain.com",
				},
			}
			cli.getServicesName = "space-drain-1"
			cli.getServicesApps = []string{"app-1", "app-2"}

			ok, err := c.DeleteDrainAndUser("space-guid", "space-drain-1")
			Expect(ok).To(BeTrue())
			Expect(err).ShouldNot(HaveOccurred())

			Expect(cli.cliCommandArgs[0]).To(Equal([]string{
				"delete", "space-drain-1", "-f",
			}))
			Expect(cli.cliCommandArgs[1]).To(Equal([]string{
				"delete-user", "space-drain-space-guid", "-f",
			}))
		})

		It("returns error when drains cannot be fetched", func() {
			ok, err := c.DeleteDrainAndUser("bad-space-guid", "drain-2")
			Expect(ok).To(BeFalse())
			Expect(err).Should(HaveOccurred())
		})

		It("returns error when drain is not found", func() {
			ok, err := c.DeleteDrainAndUser("space-guid", "drain-3")
			Expect(ok).To(BeFalse())
			Expect(err).Should(HaveOccurred())
		})
	})
})

type stubCliConnection struct {
	plugin.CliConnection

	getServicesName string
	getServicesApps []string

	getServiceError  error
	getServicesError error

	cliCommandWithoutTerminalOutputArgs     [][]string
	cliCommandWithoutTerminalOutputResponse map[string]string

	cliCommandArgs     [][]string
	unbindServiceError error
	deleteServiceError error

	setEnvErrors map[string]error
}

func newStubCliConnection() *stubCliConnection {
	return &stubCliConnection{
		cliCommandWithoutTerminalOutputResponse: make(map[string]string),
		setEnvErrors:                            make(map[string]error),
	}
}

func (s *stubCliConnection) GetServices() ([]plugin_models.GetServices_Model, error) {
	resp := []plugin_models.GetServices_Model{
		{
			Name:             "garbage-1",
			ApplicationNames: []string{"garbage-app-1", "garbage-app-2"},
		},
		{
			Name:             s.getServicesName,
			ApplicationNames: s.getServicesApps,
		},
		{
			Name:             "garbage-2",
			ApplicationNames: []string{"garbage-app-3", "garbage-app-4"},
		},
	}

	return resp, s.getServicesError
}

func (s *stubCliConnection) CliCommandWithoutTerminalOutput(args ...string) ([]string, error) {
	s.cliCommandWithoutTerminalOutputArgs = append(
		s.cliCommandWithoutTerminalOutputArgs,
		args,
	)

	output, ok := s.cliCommandWithoutTerminalOutputResponse[strings.Join(args, " ")]
	if !ok {
		output = "{}"
	}

	var err error
	switch args[0] {
	case "set-env":
		err = s.setEnvErrors[args[2]]
	}

	return strings.Split(output, "\n"), err
}

func (s *stubCliConnection) CliCommand(args ...string) ([]string, error) {
	var err error
	switch args[0] {
	case "unbind-service":
		err = s.unbindServiceError
	case "delete-service":
		err = s.deleteServiceError
	}

	s.cliCommandArgs = append(s.cliCommandArgs, args)
	return nil, err
}

type stubCurler struct {
	URLs    []string
	methods []string
	bodies  []string
	resps   map[string]string
	errs    map[string]error
}

func newStubCurler() *stubCurler {
	return &stubCurler{
		resps: make(map[string]string),
		errs:  make(map[string]error),
	}
}

func (s *stubCurler) Curl(url, method, body string) ([]byte, error) {
	s.URLs = append(s.URLs, url)
	s.methods = append(s.methods, method)
	s.bodies = append(s.bodies, body)
	return []byte(s.resps[url]), s.errs[url]
}

var serviceInstancesJSONpage1 = `{
   "total_results": 2,
   "total_pages": 2,
   "prev_url": null,
   "next_url": "/v2/user_provided_service_instances?q=space_guid:space-guid&page:2",
   "resources": [
      {
		 "metadata": {
			"guid": "guid-1"
		 },
         "entity": {
            "name": "drain-1",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "service_bindings_url": "/v2/user_provided_service_instances/drain-1/service_bindings"
         }
      }
   ]
}`

var serviceInstancesJSONpage2 = `{
   "total_results": 2,
   "total_pages": 2,
   "prev_url": "/v2/user_provided_service_instances?q=space_guid:space-guid&page:1",
   "next_url": null,
   "resources": [
      {
		 "metadata": {
			"guid": "guid-2"
		 },
         "entity": {
            "name": "drain-2",
            "syslog_drain_url": "https://your-app2.cf-app.com?drain-type=metrics",
            "service_bindings_url": "/v2/user_provided_service_instances/drain-2/service_bindings"
         }
      }
   ]
}`

var serviceBindingsJSON1page1 = `{
   "total_results": 2,
   "total_pages": 2,
   "prev_url": null,
   "next_url": "/v2/user_provided_service_instances/drain-1/service_bindings&page:2",
   "resources": [
      {
         "entity": {
            "app_guid": "app-1",
            "service_instance_guid": "drain-1",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-1"
         }
      }
   ]
}`

var serviceBindingsJSON1page2 = `{
   "total_results": 2,
   "total_pages": 2,
   "prev_url": "/v2/user_provided_service_instances/drain-1/service_bindings&page:1",
   "next_url": null,
   "resources": [
      {
         "entity": {
            "app_guid": "app-2",
            "service_instance_guid": "drain-1",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-2"
         }
      }
   ]
}`

var serviceBindingsJSON2 = `{
   "total_results": 1,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
      {
         "entity": {
            "app_guid": "app-1",
            "service_instance_guid": "drain-2",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-1"
         }
      }
   ]
}`

var appJSONpage1 = `{
   "pagination": {
      "total_results": 2,
      "total_pages": 2,
      "next": "/v3/apps?guids=app-1,app-2,app-1&page=2",
      "previous": null
   },
   "resources": [
      {
         "guid": "app-1",
         "name": "My App One"
      }
   ]
}`

var appJSONpage2 = `{
   "pagination": {
      "total_results": 2,
      "total_pages": 2,
      "next": null,
      "previous": "/v3/apps?guids=app-1,app-2,app-1&page=1"
   },
   "resources": [
      {
         "guid": "app-2",
         "name": "My App Two"
      }
   ]
}`
