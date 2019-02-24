package main

import (
	"context"
	"fmt"
	"log"

	docker "github.com/docker/docker/client"
	"github.com/wdullaer/docker-dns-updater/dns"
	"github.com/wdullaer/docker-dns-updater/store"
)

// State is a type that serves as a container for all the state the program
// manages
// It makes the signature of functions which act on all of these easier to read
// and can act as a poor mans named parameters
type State struct {
	Config       *config
	Provider     dns.DNSProvider
	DockerClient *docker.Client
	Store        store.Store
}

// NewState returns a fully initialised application State baed on the
// configuration options
func NewState(config *config) (*State, error) {
	state := &State{
		Config: config,
	}

	// Connect to docker daemon
	log.Println("[INFO] Connecting to docker")
	dockerClient, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	state.DockerClient = dockerClient
	log.Println("[INFO] Connected to docker")

	// Create the DNSProvider
	log.Printf("[INFO] Connecting to DNS Provider: %s", config.Provider)
	provider, err := getDNSProvider(config)
	if err != nil {
		return nil, err
	}
	state.Provider = provider
	log.Printf("[INFO] Connected to DNS Provider: %s", config.Provider)

	// Create the store
	log.Printf("[INFO] Connecting to Store: %s", config.Store)
	db, err := getStore(config)
	if err != nil {
		return nil, err
	}
	state.Store = db
	log.Printf("[INFO] Connected to Store: %s", state.Config.Store)

	return state, nil
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

func getDNSProvider(config *config) (dns.DNSProvider, error) {
	switch config.Provider {
	case "cloudflare":
		return dns.NewCloudflareProvider(config.AccountName, config.AccountSecret)
	case "dryrun":
		return dns.NewDryrunProvider()
	default:
		// Since we are eagerly validating the config, this should never happen
		return nil, fmt.Errorf("Invalid provider specified: %s", config.Provider)
	}
}

func getStore(config *config) (store.Store, error) {
	switch config.Store {
	case "memory":
		return store.NewMemoryStore()
	case "boltdb":
		return store.NewBoltDBStore()
	default:
		// Since we are eagerly validating the config, this should never happen
		return nil, fmt.Errorf("Invalid store specified: %s", config.Store)
	}
}
