package main

import (
	"encoding/json"
	"log"
	"os"

	envstruct "code.cloudfoundry.org/go-envstruct"
)

type Config struct {
	SpaceID string `env:"SPACE_ID, required"`

	DrainName string `env:"DRAIN_NAME, required"`
	DrainURL  string `env:"DRAIN_URL, required"`
	DrainType string `env: "DRAIN_TYPE"`

	APIAddr  string `env:"API_ADDR, required"`
	UAAAddr  string `env:"UAA_ADDR, required"`
	ClientID string `env:"CLIENT_ID, required"`

	SkipCertVerify bool `env:"SKIP_CERT_VERIFY"`

	VCAPApplication Application
	RefreshToken    string `env:"REFRESH_TOKEN"`
}

type Application struct {
	ID string `json:"application_id"`
}

func loadConfig() Config {
	cfg := Config{
		DrainType: "all",
	}
	if err := envstruct.Load(&cfg); err != nil {
		log.Fatal(err)
	}

	//TODO: The application ID needs to come from CAPI
	va := os.Getenv("VCAP_APPLICATION")
	var app Application
	err := json.Unmarshal([]byte(va), &app)
	if err != nil {
		log.Fatal(err)
	}

	cfg.VCAPApplication = app

	return cfg
}
