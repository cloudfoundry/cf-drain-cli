package command_test

import (
	"errors"

	"code.cloudfoundry.org/cf-syslog-cli/internal/command"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Drains", func() {
	var (
		logger   *stubLogger
		cli      *stubCliConnection
		ccClient *stubCloudControllerClient
	)

	BeforeEach(func() {
		logger = &stubLogger{}
		ccClient = newStubCloudControllerClient()
		cli = newStubCliConnection()
		cli.currentSpaceGuid = "my-space-guid"
	})

	It("writes the headers", func() {
		key := "/v2/user_provided_service_instances?q=space_guid:my-space-guid"
		ccClient.resps[key] = userProvidedServiceInstancesJSON
		command.Drains(cli, ccClient, []string{}, logger)

		Expect(len(logger.printfMessages)).To(BeNumerically(">", 0))
		Expect(logger.printfMessages[0]).To(MatchRegexp(`\Aname\s+bound apps`))
	})

	It("writes the drain name in the first column", func() {
		key := "/v2/user_provided_service_instances?q=space_guid:my-space-guid"
		ccClient.resps[key] = userProvidedServiceInstancesJSON
		command.Drains(cli, ccClient, []string{}, logger)

		// Header + 2 drains
		Expect(logger.printfMessages).To(HaveLen(3))
		Expect(logger.printfMessages[1]).To(MatchRegexp(`\Adrain-1`))
		Expect(logger.printfMessages[2]).To(MatchRegexp(`\Adrain-2`))
	})

	// TODO change GUID to App Name
	It("writes the app guid in the second column", func() {
		key := "/v2/user_provided_service_instances?q=space_guid:my-space-guid"
		ccClient.resps[key] = userProvidedServiceInstancesJSON
		key = "/v2/user_provided_service_instances/my-service-1/service_bindings"
		ccClient.resps[key] = serviceBindingsJSON
		key = "/v2/user_provided_service_instances/my-service-2/service_bindings"
		ccClient.resps[key] = serviceBindingsJSON2
		command.Drains(cli, ccClient, []string{}, logger)

		// Header + 2 drains
		Expect(logger.printfMessages).To(HaveLen(3))
		Expect(logger.printfMessages[1]).To(MatchRegexp(`\Adrain-1\s+app-1,\s+app-2`))
		Expect(logger.printfMessages[2]).To(MatchRegexp(`\Adrain-2\s+app-1`))
	})

	PIt("writes the drain type in the third column", func() {
	})

	PIt("reads service instances from multiple pages", func() {
	})

	PIt("reads service bindings from multiple pages", func() {
	})

	PIt("only displays syslog services", func() {
	})

	It("gets user provided service instances for the space", func() {
		key := "/v2/user_provided_service_instances?q=space_guid:my-space-guid"
		ccClient.resps[key] = userProvidedServiceInstancesJSON
		command.Drains(cli, ccClient, []string{}, logger)

		Expect(ccClient.URLs).To(HaveLen(3))
		Expect(ccClient.URLs[0]).To(Equal(
			"/v2/user_provided_service_instances?q=space_guid:my-space-guid",
		))

		Expect(ccClient.URLs[1]).To(Equal(
			"/v2/user_provided_service_instances/my-service-1/service_bindings",
		))

		Expect(ccClient.URLs[2]).To(Equal(
			"/v2/user_provided_service_instances/my-service-2/service_bindings",
		))
	})

	It("fatally logs when failing to get current space", func() {
		cli.currentSpaceError = errors.New("no space error")

		Expect(func() {
			command.Drains(cli, ccClient, []string{}, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("no space error"))
	})

	It("fatally logs when failing to get user provided service instances", func() {
		key := "/v2/user_provided_service_instances?q=space_guid:my-space-guid"
		ccClient.respErrs[key] = errors.New("not found error")

		Expect(func() {
			command.Drains(cli, ccClient, []string{}, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("not found error"))
	})

	It("fatally logs when failing to parse JSON response from user provided service instances", func() {
		key := "/curl v2/user_provided_service_instances?q=space_guid:my-space-guid"
		ccClient.resps[key] = "no"

		Expect(func() {
			command.Drains(cli, ccClient, []string{}, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(MatchRegexp(`Failed to parse response body.*`))
	})

	PIt("fatally logs when failing to get service bindings", func() {
	})
	PIt("fatally logs when failing to parse JSON response from service bindings", func() {})
	It("expects no arguments", func() {
		Expect(func() {
			command.Drains(cli, ccClient, []string{"invalid"}, logger)
		}).To(Panic())
		Expect(logger.fatalfMessage).To(Equal("Invalid arguments, expected 0, got 1."))
	})
})

var userProvidedServiceInstancesJSON = `{
   "total_results": 2,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
      {
         "entity": {
            "name": "drain-1",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "service_bindings_url": "/v2/user_provided_service_instances/my-service-1/service_bindings"
         }
      },
      {
         "entity": {
            "name": "drain-2",
            "syslog_drain_url": "syslog://your-app2.cf-app.com",
            "service_bindings_url": "/v2/user_provided_service_instances/my-service-2/service_bindings"
         }
      }
   ]
}`

var serviceBindingsJSON = `{
   "total_results": 2,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
      {
         "entity": {
            "app_guid": "app-1",
            "service_instance_guid": "service-1",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-1"
         }
      },
      {
         "entity": {
            "app_guid": "app-2",
            "service_instance_guid": "service-1",
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
            "service_instance_guid": "service-2",
            "syslog_drain_url": "syslog://your-app.cf-app.com",
            "name": null,
            "app_url": "/v2/apps/app-1"
         }
      }
   ]
}`
