package main

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
)

type Config struct {
	HttpProxy       string `env:"HTTP_PROXY, required, report"`
	SourceID        string `env:"SOURCE_ID,                 report"`
	SourceHostname  string `env:"SOURCE_HOSTNAME, required, report"`
	IncludeServices bool   `env:"INCLUDE_SERVICES, report"`

	SyslogURL *url.URL `env:"SYSLOG_URL, required, report"`

	SkipCertVerify bool `env:"SKIP_CERT_VERIFY, report"`

	UpdateInterval time.Duration `env:"UPDATE_INTERVAL, report"`
	DialTimeout    time.Duration `env:"DIAL_TIMEOUT,    report"`
	IOTimeout      time.Duration `env:"IO_TIMEOUT,      report"`
	KeepAlive      time.Duration `env:"KEEP_ALIVE,      report"`
	Vcap           VCap          `env:"VCAP_APPLICATION,        required"`

	ShardID string // Comes from the VCAP application ID
}

func LoadConfig() Config {
	cfg := Config{
		UpdateInterval: 30 * time.Second,
		SkipCertVerify: false,
		KeepAlive:      10 * time.Second,
		DialTimeout:    5 * time.Second,
		IOTimeout:      time.Minute,
	}
	if err := envstruct.Load(&cfg); err != nil {
		log.Fatalf("failed to load config from environment: %s", err)
	}

	cfg.ShardID = cfg.Vcap.AppID

	return cfg
}

type VCap struct {
	AppID     string `json:"application_id"`
	API       string `json:"cf_api"`
	SpaceGUID string `json:"space_id"`

	// Derived from VcapApplication
	RLPAddr string
}

func (v *VCap) UnmarshalEnv(data string) error {
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return err
	}
	v.RLPAddr = strings.Replace(v.API, "https://api", "http://log-stream", 1)
	v.API = strings.Replace(v.API, "https", "http", 1)

	return nil
}
