package cloudcontroller_test

import (
	"errors"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DrainsClient", func() {
	var (
		curler *stubCurler
		c      *cloudcontroller.DrainsClient
		key    string
	)

	BeforeEach(func() {
		curler = newStubCurler()
		c = cloudcontroller.NewDrainsClient(curler)
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
					Expect(d[0].Apps).To(Equal([]string{"My App One", "My App Two"}))
					Expect(d[0].Type).To(Equal("logs"))

					Expect(d[1].Name).To(Equal("drain-2"))
					Expect(d[1].Apps).To(Equal([]string{"My App One"}))
					Expect(d[1].Type).To(Equal("metrics"))
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
})

type stubCurler struct {
	URLs  []string
	resps map[string]string
	errs  map[string]error
}

func newStubCurler() *stubCurler {
	return &stubCurler{
		resps: make(map[string]string),
		errs:  make(map[string]error),
	}
}

func (s *stubCurler) Curl(URL string) ([]byte, error) {
	s.URLs = append(s.URLs, URL)
	return []byte(s.resps[URL]), s.errs[URL]
}

var serviceInstancesJSONpage1 = `{
   "total_results": 2,
   "total_pages": 2,
   "prev_url": null,
   "next_url": "/v2/user_provided_service_instances?q=space_guid:space-guid&page:2",
   "resources": [
      {
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
