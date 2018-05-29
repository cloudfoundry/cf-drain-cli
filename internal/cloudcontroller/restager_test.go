package cloudcontroller_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var _ = Describe("Restager", func() {
	var (
		r          *cloudcontroller.Restager
		ac         *spyAuthCurler
		stubLogger *stubLogger
	)

	BeforeEach(func() {
		ac = &spyAuthCurler{}
		stubLogger = newStubLogger()
		r = cloudcontroller.NewRestager("app-guid", ac, stubLogger)
	})

	It("saves new refresh token and restage", func() {
		r.SaveAndRestage("new-refresh-token")

		Expect(ac.urls).To(HaveLen(2))

		Expect(ac.urls[0]).To(Equal("/v3/apps/app-guid/environment_variables"))
		Expect(ac.methods[0]).To(Equal("PATCH"))
		Expect(ac.bodies[0]).To(MatchJSON(`{
			     "var": {
					 "REFRESH_TOKEN": "new-refresh-token"
				}
		}`))

		Expect(ac.urls[1]).To(Equal("/v2/apps/app-guid/restage"))
		Expect(ac.methods[1]).To(Equal("POST"))
		Expect(ac.bodies[1]).To(Equal(""))
	})

	It("panics if unable to save REFRESH_TOKEN to cloud controller", func() {
		ac.errs = []error{errors.New("CAPI is down")}
		Expect(func() { r.SaveAndRestage("some-token") }).To(Panic())
		Expect(stubLogger.called).To(Equal(1))
	})

	It("panics if unable to restage app", func() {
		ac.errs = []error{nil, errors.New("CAPI is down")}
		Expect(func() { r.SaveAndRestage("some-token") }).To(Panic())
		Expect(stubLogger.called).To(Equal(1))
	})
})
