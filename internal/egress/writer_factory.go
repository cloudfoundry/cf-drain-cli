package egress

import (
	"io"
	"log"
	"net/url"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
)

// Write is the interface for all diode writers.
type Writer interface {
	Write(*loggregator_v2.Envelope) error
}

// WriteCloser is the interface for all syslog writers.
type WriteCloser interface {
	Writer
	io.Closer
}

func NewWriter(sourceHost string, url *url.URL, netConf NetworkConfig, log *log.Logger) WriteCloser {
	binding := URLBinding{
		URL:      url,
		Hostname: sourceHost,
	}

	switch url.Scheme {
	case "syslog":
		return NewTCPWriter(&binding, netConf)
	case "syslog-tls":
		return NewTLSWriter(&binding, netConf)
	case "https":
		return NewHTTPSWriter(&binding, netConf)
	default:
		log.Fatalf("unable to create writer for scheme: %s", url.Scheme)
	}

	return nil
}
