package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
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
