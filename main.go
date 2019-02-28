package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Load initial configuration
	configuration := parseFlags()
	logger, err := getLogger(configuration)
	if err != nil {
		log.Fatalf("[FATAL] Failed to instantiate logger: %s", err)
	}

	// Validate the configuration
	if errs := configuration.Validate(); len(errs) != 0 {
		logger.Fatalw("Invalid configuration values", "errors", errs)
	}
	logger.Infow("Using configuration", "configuration", configuration)

	// Initialize application state
	state, err := NewState(configuration, logger)
	if err != nil {
		logger.Fatalw("Failed to initialize application", "err", err)
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
				state.Logger.Errorw("Failed to process docker event", "err", err)
			}
		case err := <-errorChan:
			state.Logger.Fatalw("Received a docker error", "err", err)
			break main
		case sig := <-signalChan:
			state.Logger.Infow("Received signal to terminate", "sig", sig)
			break main
		}
	}
}
