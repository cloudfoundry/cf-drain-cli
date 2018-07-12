package egress

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"code.cloudfoundry.org/cf-drain-cli/internal/egress/config"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
)

// NetworkTimeoutConfig stores various timeout values.
type NetworkConfig struct {
	Keepalive      time.Duration
	DialTimeout    time.Duration
	WriteTimeout   time.Duration
	SkipCertVerify bool
}

type HTTPSWriter struct {
	hostname string
	url      *url.URL
	client   *http.Client
}

func NewHTTPSWriter(
	binding *URLBinding,
	netConf NetworkConfig,
) WriteCloser {

	client := httpClient(netConf)

	return &HTTPSWriter{
		url:      binding.URL,
		hostname: binding.Hostname,
		client:   client,
	}
}

func (w *HTTPSWriter) Write(env *loggregator_v2.Envelope) error {
	msgs := generateRFC5424Messages(env, w.hostname, env.SourceId)
	for _, msg := range msgs {
		b, err := msg.MarshalBinary()
		if err != nil {
			return err
		}

		resp, err := w.client.Post(w.url.String(), "text/plain", bytes.NewBuffer(b))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return fmt.Errorf("Syslog Writer: Post responded with %d status code", resp.StatusCode)
		}

		io.Copy(ioutil.Discard, resp.Body)
	}

	return nil
}

func (*HTTPSWriter) Close() error {
	return nil
}

func httpClient(netConf NetworkConfig) *http.Client {
	tlsConfig := config.NewTLSConfig()
	tlsConfig.InsecureSkipVerify = netConf.SkipCertVerify

	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   netConf.DialTimeout,
			KeepAlive: netConf.Keepalive,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       tlsConfig,
	}

	return &http.Client{
		Transport: tr,
		Timeout:   60 * time.Second,
	}
}
