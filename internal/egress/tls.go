package egress

import (
	"crypto/tls"
	"net"
)

// TLSWriter represents a syslog writer that connects over unencrypted TCP.
type TLSWriter struct {
	TCPWriter
}

func NewTLSWriter(
	binding *URLBinding,
	netConf NetworkConfig,
) WriteCloser {

	dialer := &net.Dialer{
		Timeout:   netConf.DialTimeout,
		KeepAlive: netConf.Keepalive,
	}
	df := func(addr string) (net.Conn, error) {
		return tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			InsecureSkipVerify: netConf.SkipCertVerify,
		})
	}

	w := &TLSWriter{
		TCPWriter{
			url:          binding.URL,
			hostname:     binding.Hostname,
			writeTimeout: netConf.WriteTimeout,
			dialFunc:     df,
			scheme:       "syslog-tls",
		},
	}

	return w
}
