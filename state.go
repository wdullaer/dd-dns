package main

import (
	"context"
	"fmt"

	docker "github.com/docker/docker/client"
	"github.com/wdullaer/dd-dns/dns"
	"github.com/wdullaer/dd-dns/store"
	"go.uber.org/zap"
)

// State is a type that serves as a container for all the state the program
// manages
// It makes the signature of functions which act on all of these easier to read
// and can act as a poor mans named parameters
type State struct {
	Config       *config
	Provider     dns.Provider
	DockerClient *docker.Client
	Store        store.Store
	Logger       *zap.SugaredLogger
}

// NewState returns a fully initialized application State baed on the
// configuration options
func NewState(config *config, logger *zap.SugaredLogger) (*State, error) {
	state := &State{
		Config: config,
		Logger: logger,
	}

	// Connect to docker daemon
	state.Logger.Infow("Connecting to docker")
	dockerClient, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	state.DockerClient = dockerClient
	state.Logger.Infow("Connected to docker")

	// Create the Provider
	state.Logger.Infow("Connecting to DNS Provider", "provider", config.Provider)
	provider, err := getDNSProvider(config, logger)
	if err != nil {
		return nil, err
	}
	state.Provider = provider
	state.Logger.Infow("Connected to DNS Provider", "provider", config.Provider)

	// Create the store
	state.Logger.Infow("Connecting to Store", "store", config.Store)
	db, err := getStore(config, logger)
	if err != nil {
		return nil, err
	}
	state.Store = db
	state.Logger.Infow("Connected to Store", "store", state.Config.Store)

	return state, nil
}

func getLogger(config *config) (*zap.SugaredLogger, error) {
	var logger *zap.Logger
	var err error

	if config.DebugLogger {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		return nil, err
	}
	return logger.Named("dd-dns").Sugar(), nil
}

func getDockerClient() (*docker.Client, error) {
	dockerClient, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		return nil, err
	}

	dockerPing, err := dockerClient.Ping(context.Background())
	if err != nil {
		return nil, err
	}
	dockerClient.NegotiateAPIVersionPing(dockerPing)

	return dockerClient, nil
}

func getDNSProvider(config *config, logger *zap.SugaredLogger) (dns.Provider, error) {
	switch config.Provider {
	case providerCloudflare:
		return dns.NewCloudflareProvider(config.AccountName, config.AccountSecret, logger)
	case providerDryrun:
		return dns.NewDryrunProvider(logger)
	default:
		// Since we are eagerly validating the config, this should never happen
		return nil, fmt.Errorf("invalid provider specified: %s", config.Provider)
	}
}

func getStore(config *config, logger *zap.SugaredLogger) (store.Store, error) {
	switch config.Store {
	case storeMemory:
		return store.NewMemoryStore(logger)
	case storeBoltdb:
		return store.NewBoltDBStore(logger, config.DataDirectory)
	default:
		// Since we are eagerly validating the config, this should never happen
		return nil, fmt.Errorf("invalid store specified: %s", config.Store)
	}
}
