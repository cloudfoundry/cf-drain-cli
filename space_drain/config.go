package main

import (
	"log"

	envstruct "code.cloudfoundry.org/go-envstruct"
)

type Config struct {
	SpaceID string `env:"SPACE_ID, required"`
	// DrainType string `env:"DRAIN_TYPE, required"`
	DrainName string `env:"DRAIN_NAME, required"`
	DrainURL  string `env:"DRAIN_URL, required"`
	DrainType string `env: "DRAIN_TYPE"`

	APIAddr      string `env:"API_ADDR, required"`
	UAAAddr      string `env:"UAA_ADDR, required"`
	ClientID     string `env:"CLIENT_ID, required"`
	ClientSecret string `env:"CLIENT_SECRET"`

	Username string `env:"USERNAME, required"`
	Password string `env:"PASSWORD, required"`

	SkipCertVerify bool `env:"SKIP_CERT_VERIFY"`
}

func loadConfig() Config {
	cfg := Config{
		DrainType: "all",
	}
	if err := envstruct.Load(&cfg); err != nil {
		log.Fatal(err)
	}

	return cfg
}
