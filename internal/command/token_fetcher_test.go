package command_test

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-drain-cli/internal/command"
)

var _ = Describe("TokenFetcher", func() {
	var (
		f          *command.TokenFetcher
		configPath string
	)

	Context("valid config", func() {
		BeforeEach(func() {
			configFile, err := ioutil.TempFile("", "token_fetcher_tests")
			writeRefreshToken(configFile)
			configFile.Close()

			Expect(err).ToNot(HaveOccurred())
			configPath = configFile.Name()

			f = command.NewTokenFetcher(configPath)
		})

		AfterEach(func() {
			os.Remove(configPath)
		})

		It("returns the token", func() {
			token, err := f.RefreshToken()
			Expect(err).ToNot(HaveOccurred())
			Expect(token).To(Equal("some-token"))
		})
	})

	It("returns err if the file can't be read from", func() {
		f = command.NewTokenFetcher("invalid")
		_, err := f.RefreshToken()
		Expect(err).To(HaveOccurred())
	})

	Context("invalid config file", func() {
		BeforeEach(func() {
			configFile, err := ioutil.TempFile("", "token_fetcher_tests")
			writeInvalidRefreshToken(configFile)
			configFile.Close()

			Expect(err).ToNot(HaveOccurred())
			configPath = configFile.Name()

			f = command.NewTokenFetcher(configPath)
		})

		AfterEach(func() {
			os.Remove(configPath)
		})

		It("returns err if config file isn't valid json", func() {
			_, err := f.RefreshToken()
			Expect(err).To(HaveOccurred())
		})

	})
})

func writeRefreshToken(w io.Writer) {
	_, err := io.Copy(w, strings.NewReader(`{
     "RefreshToken": "some-token"
    }`))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func writeInvalidRefreshToken(w io.Writer) {
	_, err := io.Copy(w, strings.NewReader(`{
     "RefreshToken": "some-token"
    `))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}
