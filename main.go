package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	provider      = flag.String("provider", os.Getenv("PROVIDER"), "The DNS provider to register the domain names with")
	accountName   = flag.String("account-name", os.Getenv("ACCOUNT_NAME"), "The account-name (or equivalent) to be used for authenticating with the DNS provider")
	accountSecret = flag.String("account-secret", os.Getenv("ACCOUNT_SECRET"), "The account-secret (or equivalent) to be used for authenticating with the DNS provider")
	dnsContent    = flag.String("dns-content", os.Getenv("DNS_CONTENT"), "The IP address to be added to the DNS content (default: `container`, oneOf: [`container`, `<ipv4>`]")
	dockerLabel   = flag.String("docker-label", os.Getenv("DOCKER_LABEL"), "The docker label that contains the domain name")
	storeName     = flag.String("store", os.Getenv("STORE"), "The store implemenation that persists the internal state")
)

func main() {
	// Load and validate the configuration
	flag.Parse()
	configuration := &config{
		Provider:      *provider,
		AccountName:   *accountName,
		AccountSecret: *accountSecret,
		DNSContent:    *dnsContent,
		DockerLabel:   *dockerLabel,
		Store:         *storeName,
	}
	if errs := configuration.Validate(); len(errs) != 0 {
		for i := range errs {
			log.Printf("[FATAL] Invalid configuration value: %s", errs[i])
		}
		os.Exit(1)
	}
	log.Printf("[INFO] Using configuration: %s", configuration)

	// Initialize application state
	state, err := NewState(configuration)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize application: %s", err)
	}
	defer state.Store.CleanUp()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// TODO: maybe regularly sync with docker
	syncDNSWithDocker(state)

	eventChan, errorChan := makeDockerChannels(state.DockerClient, state.Config)
main:
	for {
		select {
		case event := <-eventChan:
			err := processDockerEvent(event, state)
			if err != nil {
				log.Fatalf("[FATAL] Failed to process docker event: %s", err)
			}
		case err := <-errorChan:
			log.Fatalf("[FATAL] Received a docker error: %s", err)
			break main
		case sig := <-signalChan:
			log.Printf("[INFO] Received signal to terminate: %s", sig)
			break main
		}
	}
}
