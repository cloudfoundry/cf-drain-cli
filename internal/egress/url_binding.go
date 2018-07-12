package egress

// URLBinding associates a particular application with a syslog URL. The
import (
	"context"
	"net/url"
)

// application is identified by AppID and Hostname. The syslog URL is
// identified by URL.
type URLBinding struct {
	Context  context.Context
	Hostname string
	URL      *url.URL
}

// Scheme is a convenience wrapper around the *url.URL Scheme field
func (u *URLBinding) Scheme() string {
	return u.URL.Scheme
}
