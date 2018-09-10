package egress

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/rfc5424"
)

// gaugeStructuredDataID contains the registered enterprise ID for the Cloud
// Foundry Foundation.
// See: https://www.iana.org/assignments/enterprise-numbers/enterprise-numbers
const (
	gaugeStructuredDataID   = "gauge@47450"
	counterStructuredDataID = "counter@47450"
)

// DialFunc represents a method for creating a connection, either TCP or TLS.
type DialFunc func(addr string) (net.Conn, error)

// TCPWriter represents a syslog writer that connects over unencrypted TCP.
// This writer is not meant to be used from multiple goroutines. The same
// goroutine that calls `.Write()` should be the one that calls `.Close()`.
type TCPWriter struct {
	url          *url.URL
	hostname     string
	dialFunc     DialFunc
	writeTimeout time.Duration
	scheme       string
	conn         net.Conn
}

// NewTCPWriter creates a new TCP syslog writer.
func NewTCPWriter(
	binding *URLBinding,
	netConf NetworkConfig,
) WriteCloser {
	dialer := &net.Dialer{
		Timeout:   netConf.DialTimeout,
		KeepAlive: netConf.Keepalive,
	}
	df := func(addr string) (net.Conn, error) {
		return dialer.Dial("tcp", addr)
	}

	w := &TCPWriter{
		url:          binding.URL,
		hostname:     binding.Hostname,
		writeTimeout: netConf.WriteTimeout,
		dialFunc:     df,
		scheme:       "syslog",
	}

	return w
}

func (w *TCPWriter) connection() (net.Conn, error) {
	if w.conn == nil {
		return w.connect()
	}
	return w.conn, nil
}

func (w *TCPWriter) connect() (net.Conn, error) {
	conn, err := w.dialFunc(w.url.Host)
	if err != nil {
		return nil, err
	}
	w.conn = conn

	log.Printf("created conn to syslog drain: %s", w.url.Host)

	return conn, nil
}

// Close tears down any active connections to the drain and prevents reconnect.
func (w *TCPWriter) Close() error {
	if w.conn != nil {
		err := w.conn.Close()
		w.conn = nil

		return err
	}

	return nil
}

func generateRFC5424Messages(
	env *loggregator_v2.Envelope,
	hostname string,
	appID string,
) []rfc5424.Message {
	hostname = fmt.Sprintf("%s.%s", hostname, env.GetTags()["hostname_suffix"])

	switch env.GetMessage().(type) {
	case *loggregator_v2.Envelope_Log:
		return []rfc5424.Message{
			{
				Priority:  generatePriority(env.GetLog().Type),
				Timestamp: time.Unix(0, env.GetTimestamp()).UTC(),
				Hostname:  hostname,
				AppName:   appID,
				ProcessID: generateProcessID(
					env.Tags["source_type"],
					env.InstanceId,
				),
				Message: appendNewline(removeNulls(env.GetLog().Payload)),
			},
		}
	case *loggregator_v2.Envelope_Gauge:
		gauges := make([]rfc5424.Message, 0, 5)

		for name, g := range env.GetGauge().GetMetrics() {
			gauges = append(gauges, rfc5424.Message{
				Priority:  rfc5424.Info + rfc5424.User,
				Timestamp: time.Unix(0, env.GetTimestamp()).UTC(),
				Hostname:  hostname,
				AppName:   appID,
				ProcessID: fmt.Sprintf("[%s]", env.InstanceId),
				Message:   []byte("\n"),
				StructuredData: []rfc5424.StructuredData{
					{
						ID: gaugeStructuredDataID,
						Parameters: []rfc5424.SDParam{
							{
								Name:  "name",
								Value: name,
							},
							{
								Name:  "value",
								Value: strconv.FormatFloat(g.GetValue(), 'g', -1, 64),
							},
							{
								Name:  "unit",
								Value: g.GetUnit(),
							},
						},
					},
				},
			})
		}

		return gauges
	case *loggregator_v2.Envelope_Counter:
		return []rfc5424.Message{
			{
				Priority:  rfc5424.Info + rfc5424.User,
				Timestamp: time.Unix(0, env.GetTimestamp()).UTC(),
				Hostname:  hostname,
				AppName:   appID,
				ProcessID: fmt.Sprintf("[%s]", env.InstanceId),
				Message:   []byte("\n"),
				StructuredData: []rfc5424.StructuredData{
					{
						ID: counterStructuredDataID,
						Parameters: []rfc5424.SDParam{
							{
								Name:  "name",
								Value: env.GetCounter().GetName(),
							},
							{
								Name:  "total",
								Value: fmt.Sprint(env.GetCounter().GetTotal()),
							},
							{
								Name:  "delta",
								Value: fmt.Sprint(env.GetCounter().GetDelta()),
							},
						},
					},
				},
			},
		}
	default:
		return []rfc5424.Message{}
	}
}

// Write writes an envelope to the syslog drain connection.
func (w *TCPWriter) Write(env *loggregator_v2.Envelope) error {
	msgs := generateRFC5424Messages(env, w.hostname, env.SourceId)
	conn, err := w.connection()
	if err != nil {
		return err
	}

	for _, msg := range msgs {
		conn.SetWriteDeadline(time.Now().Add(w.writeTimeout))
		_, err = msg.WriteTo(conn)
		if err != nil {
			_ = w.Close()

			return err
		}
	}

	return nil
}

func removeNulls(msg []byte) []byte {
	return bytes.Replace(msg, []byte{0}, nil, -1)
}

func appendNewline(msg []byte) []byte {
	if !bytes.HasSuffix(msg, []byte("\n")) {
		msg = append(msg, byte('\n'))
	}
	return msg
}

func generatePriority(logType loggregator_v2.Log_Type) rfc5424.Priority {
	switch logType {
	case loggregator_v2.Log_OUT:
		return rfc5424.Info + rfc5424.User
	case loggregator_v2.Log_ERR:
		return rfc5424.Error + rfc5424.User
	default:
		return rfc5424.Priority(-1)
	}
}

func generateProcessID(sourceType, sourceInstance string) string {
	sourceType = strings.ToUpper(sourceType)
	if sourceInstance != "" {
		return fmt.Sprintf("[%s/%s]",
			strings.Replace(sourceType, " ", "-", -1),
			sourceInstance,
		)
	}

	return fmt.Sprintf("[%s]", sourceType)
}
