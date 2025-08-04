package main

import (
	"flag"
	"log"

	"code.cloudfoundry.org/cf-tcp-router/config"
)

var (
	configFile         string
	enableCertCreation bool
)

func main() {
	flag.StringVar(&configFile, "config", "", "Configuration File")

	// enableCertCreation is a special-case flag that allows the TCP router to generate
	// certificates and keys when frontend_tls certificates are defined in tcp_router.yml
	flag.BoolVar(&enableCertCreation, "enable-cert-creation", false, "Enables creation certs and keys")
	flag.Parse()

	_, err := config.New(configFile, enableCertCreation)
	if err != nil {
		log.Fatal("failed-to-load-config: ", err)
	}
	log.Print("config-loaded-successfully")
}
