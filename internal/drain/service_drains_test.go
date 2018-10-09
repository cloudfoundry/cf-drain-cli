package drain_test

import (
	"errors"
	"fmt"
	"strings"

	"code.cloudfoundry.org/cf-drain-cli/internal/drain"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ListDrainsClient", func() {
	var (
		curler *stubCurler
		c      *drain.ServiceDrainLister
		key    string
	)

	BeforeEach(func() {
		curler = newStubCurler()
		c = drain.NewServiceDrainLister(
			curler,
			drain.WithServiceDrainAppBatchLimit(3),
		)
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

	It("sets UseAgent to true if the drain url scheme has a '-v3' suffix", func() {
		key = "/v2/user_provided_service_instances?q=space_guid:space-guid"
		curler.resps[key] = useAgentServiceInstanceJSON

		key = "/v2/user_provided_service_instances/drain-2/service_bindings"
		curler.resps[key] = useAgentServiceBindingJSON

		key = "/v3/apps?guids=app-1"
		curler.resps[key] = appJSONcall1

		d, err := c.Drains("space-guid")
		Expect(err).ToNot(HaveOccurred())
		Expect(d).To(HaveLen(1))

		Expect(d[0].DrainURL).To(Equal("syslog-v3://your-app.cf-app.com"))
		Expect(d[0].UseAgent).To(BeTrue())
	})

	Context("requesting more than apps than the batch limit", func() {
		BeforeEach(func() {
			key = "/v2/user_provided_service_instances?q=space_guid:space-guid"
			curler.resps[key] = serviceInstancesJSON2

			key = "/v2/user_provided_service_instances/drain-2/service_bindings"
			curler.resps[key] = serviceBindingsJSON3

			key = "/v3/apps?guids=app-1,app-2,app-3"
			curler.resps[key] = appJSONcall1

			key = "/v3/apps?guids=app-4"
			curler.resps[key] = appJSONcall2
		})

		It("returns every drain", func() {
			d, err := c.Drains("space-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(HaveLen(1))

			Expect(d[0].Name).To(Equal("drain-2"))
			Expect(d[0].Guid).To(Equal("guid-2"))
			Expect(d[0].Apps).To(Equal([]string{"My App One", "My App Two", "My App Three", "My App Four"}))
			Expect(d[0].AppGuids).To(Equal([]string{"app-1", "app-2", "app-3", "app-4"}))
			Expect(d[0].Type).To(Equal("logs"))
			Expect(d[0].DrainURL).To(Equal("syslog://your-app.cf-app.com"))

			Expect(curler.URLs[2:]).To(Equal([]string{
				"/v3/apps?guids=app-1,app-2,app-3",
				"/v3/apps?guids=app-4",
			}))

			// 8 => 2 service fetch + (2 app fetches) +  (4 app name fetches)
			Expect(curler.methods).To(ConsistOf("GET", "GET", "GET", "GET"))
			Expect(curler.bodies).To(ConsistOf("", "", "", ""))
		})
	})

	Context("when requesting service instances succeeds", func() {
		BeforeEach(func() {
			key = "/v2/user_provided_service_instances?q=space_guid:space-guid"
			curler.resps[key] = serviceInstancesJSONpage1
			key = "/v2/user_provided_service_instances?q=space_guid:space-guid&page:2"
			curler.resps[key] = serviceInstancesJSONpage2
		})

		Context("requesting service bindings succeeds", func() {
			BeforeEach(func() {
				key = "/v2/user_provided_service_instances/drain-1/service_bindings"
				curler.resps[key] = serviceBindingsJSON1page1
				key = "/v2/user_provided_service_instances/drain-1/service_bindings&page:2"
				curler.resps[key] = serviceBindingsJSON1page2
				key = "/v2/user_provided_service_instances/drain-2/service_bindings"
				curler.resps[key] = serviceBindingsJSON2
			})

			Context("requesting app names succeeds", func() {
				BeforeEach(func() {
					key = "/v3/apps?guids=app-1,app-2"
					curler.resps[key] = appJSONpage1

					key = "/v3/apps?guids=app-1,app-2&page=2"
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

					Expect(d[1].Name).To(Equal("drain-2"))
					Expect(d[1].Guid).To(Equal("guid-2"))
					Expect(d[1].Apps).To(Equal([]string{"My App One"}))
					Expect(d[1].AppGuids).To(Equal([]string{"app-1"}))
					Expect(d[1].Type).To(Equal("metrics"))
					Expect(d[1].DrainURL).To(Equal("https://your-app2.cf-app.com?drain-type=metrics"))

					// 7 => 2 service fetch + (2 app fetches) +  (3 app name fetches)
					Expect(curler.methods).To(ConsistOf("GET", "GET", "GET", "GET", "GET", "GET", "GET"))
					Expect(curler.bodies).To(ConsistOf("", "", "", "", "", "", ""))
				})
			})

			It("returns the error if requesting the apps fails", func() {
				key = "/v3/apps?guids=app-1,app-2"
				curler.errs[key] = errors.New("some error")

				_, err := c.Drains("space-guid")
				Expect(err).To(MatchError("some error"))
			})

			It("returns the error if unmarshalling the apps fails", func() {
				key = "/v3/apps?guids=app-1,app-2"
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
})

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

func (s *stubCurler) Curl(URL, method, body string) ([]byte, error) {
	URL = strings.Replace(URL, "%2C", ",", -1)

	s.URLs = append(s.URLs, URL)
	s.methods = append(s.methods, method)
	s.bodies = append(s.bodies, body)

	if s.errs[URL] != nil {
		return nil, s.errs[URL]
	}

	resp, ok := s.resps[URL]
	if !ok {
		panic(fmt.Sprintf("unhandled endpoint in stubCurler: %s", URL))
	}

	return []byte(resp), s.errs[URL]
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

var serviceInstancesJSON2 = `{
   "total_results": 1,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
      {
		 "metadata": {
			"guid": "guid-2"
		 },
         "entity": {
            "name": "drain-2",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
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

var serviceBindingsJSON3 = `{
   "total_results": 6,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
      {
         "entity": {
            "app_guid": "app-1",
            "service_instance_guid": "drain-1",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-1"
         }
      },
      {
         "entity": {
            "app_guid": "app-2",
            "service_instance_guid": "drain-2",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-2"
         }
      },
      {
         "entity": {
            "app_guid": "app-3",
            "service_instance_guid": "drain-3",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-3"
         }
      },
      {
         "entity": {
            "app_guid": "app-4",
            "service_instance_guid": "drain-4",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-4"
         }
      },
      {
         "entity": {
            "app_guid": "app-4",
            "service_instance_guid": "drain-5",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-4"
         }
      },
      {
         "entity": {
            "app_guid": "app-4",
            "service_instance_guid": "drain-6",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-4"
         }
      }
   ]
}`

var appJSONcall1 = `{
   "pagination": {
      "total_results": 3,
      "total_pages": 1,
      "next": null,
      "previous": null
   },
   "resources": [
      {
         "guid": "app-1",
         "name": "My App One"
      },
      {
         "guid": "app-2",
         "name": "My App Two"
      },
      {
         "guid": "app-3",
         "name": "My App Three"
      }
   ]
}`

var appJSONcall2 = `{
   "pagination": {
      "total_results": 1,
      "total_pages": 1,
      "next": null,
      "previous": null
   },
   "resources": [
      {
         "guid": "app-4",
         "name": "My App Four"
      }
   ]
}`

var appJSONpage1 = `{
   "pagination": {
      "total_results": 2,
      "total_pages": 2,
      "next": "/v3/apps?guids=app-1,app-2&page=2",
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
      "previous": "/v3/apps?guids=app-1,app-2&page=1"
   },
   "resources": [
      {
         "guid": "app-2",
         "name": "My App Two"
      }
   ]
}`

var useAgentServiceBindingJSON = `{
   "total_results": 1,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
	  {
		 "entity": {
			"app_guid": "app-1",
			"service_instance_guid": "drain-2",
			"syslog_drain_url": "syslog-v3://your-app.cf-app.com",
			"name": null,
			"app_url": "/v2/apps/app-1"
		 }
	  }
   ]
}`

var useAgentServiceInstanceJSON = `{
   "total_results": 1,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
	  {
		 "metadata": {
			"guid": "guid-2"
		 },
		 "entity": {
			"name": "drain-2",
			"syslog_drain_url": "syslog-v3://your-app.cf-app.com",
			"service_bindings_url": "/v2/user_provided_service_instances/drain-2/service_bindings"
		 }
	  }
   ]
}`
