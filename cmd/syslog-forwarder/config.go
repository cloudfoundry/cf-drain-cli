package main

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
	"github.com/nu7hatch/gouuid"
)

type Config struct {
	SourceHostname string `env:"SOURCE_HOSTNAME, required, report"`
	GroupName      string `env:"GROUP_NAME, required, report"`

	SyslogURL *url.URL `env:"SYSLOG_URL, required, report"`

	SkipCertVerify bool `env:"SKIP_CERT_VERIFY, report"`

	DialTimeout time.Duration `env:"DIAL_TIMEOUT, report"`
	IOTimeout   time.Duration `env:"IO_TIMEOUT, report"`
	KeepAlive   time.Duration `env:"KEEP_ALIVE, report"`

	// Derived from VcapApplication
	LogCacheHost string
}

func LoadConfig() Config {
	defaultGroup, err := uuid.NewV4()
	if err != nil {
		log.Fatalf("unable to generate uuid: %s", err)
	}

	cfg := Config{
		SkipCertVerify: false,
		GroupName:      defaultGroup.String(),
		KeepAlive:      10 * time.Second,
		DialTimeout:    5 * time.Second,
		IOTimeout:      time.Minute,
	}
	if err := envstruct.Load(&cfg); err != nil {
		log.Fatalf("failed to load config from environment: %s", err)
	}

	cfg.LogCacheHost = getLogCacheHost()

	return cfg
}

func getLogCacheHost() string {
	vcapEnv := os.Getenv("VCAP_APPLICATION")
	if vcapEnv == "" {
		log.Fatalf("failed to load VCAP_APPLICATION from envrionment")
	}

	var vcap struct {
		API string `json:"cf_api"`
	}

	err := json.Unmarshal([]byte(vcapEnv), &vcap)
	if err != nil {
		log.Fatalf("failed to unmarshal VCAP_APPLICATION")
	}

	return strings.Replace(vcap.API, "https://api", "http://log-cache", 1)
}
