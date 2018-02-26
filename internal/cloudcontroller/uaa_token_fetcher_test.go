package cloudcontroller_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/cloudcontroller"
)

var _ = Describe("UaaTokenFetcher", func() {
	var (
		doer *spyDoer
		f    *cloudcontroller.UAATokenFetcher
	)

	BeforeEach(func() {
		doer = newSpyDoer()
		doer.respBody = `{"access_token": "some-token", "token_type": "bearer"}`
		f = cloudcontroller.NewUAATokenFetcher(
			"https://uaa.system-domain.com",
			"some-id",
			"some-secret",
			"some-user",
			"some-password",
			doer,
		)
	})

	It("requests a token from the UAA", func() {
		token, err := f.Token()

		Expect(err).ToNot(HaveOccurred())
		Expect(token).To(Equal("bearer some-token"))
		Expect(doer.URLs).To(ConsistOf("https://some-id:some-secret@uaa.system-domain.com/oauth/token"))
		Expect(doer.methods).To(ConsistOf("POST"))
		Expect(doer.headers).To(ConsistOf(
			And(
				HaveKeyWithValue("Accept", []string{"application/json"}),
				HaveKeyWithValue("Content-Type", []string{"application/x-www-form-urlencoded"}),
			),
		))

		Expect(doer.bodies).To(ConsistOf(
			"grant_type=password&response_type=token&username=some-user&password=some-password",
		))
	})

	It("returns an error if UAA fails", func() {
		doer.err = errors.New("some-uaa-error")
		_, err := f.Token()
		Expect(err).To(MatchError("some-uaa-error"))
	})

	It("returns an error if it fails to decode JSON", func() {
		doer.respBody = `invalid`
		_, err := f.Token()
		Expect(err).To(HaveOccurred())
	})
})
