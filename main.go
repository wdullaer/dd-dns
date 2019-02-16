package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/wdullaer/docker-dns-updater/dns"
	"github.com/wdullaer/docker-dns-updater/store"
	"github.com/wdullaer/docker-dns-updater/types"

	dt "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	docker "github.com/docker/docker/client"
)

var (
	provider      = flag.String("provider", os.Getenv("PROVIDER"), "The DNS provider to register the domain names with")
	accountName   = flag.String("account-name", os.Getenv("ACCOUNT_NAME"), "The account-name (or equivalent) to be used for authenticating with the DNS provider")
	accountSecret = flag.String("account-secret", os.Getenv("ACCOUNT_SECRET"), "The account-secret (or equivalent) to be used for authenticating with the DNS provider")
	dnsContent    = flag.String("dns-content", os.Getenv("DNS_CONTENT"), "The IP address to be added to the DNS content (default: `container`, oneOf: [`container`, `<ipv4>`]")
	dockerLabel   = flag.String("docker-label", os.Getenv("DOCKER_LABEL"), "The docker label that contains the domain name")
	storeName     = flag.String("store", os.Getenv("STORE"), "The store implemenation that persists the internal state")
)

type state struct {
	Config       *config
	Provider     dns.DNSProvider
	DockerClient *docker.Client
	Store        store.Store
}

func main() {
	log.Println("[INFO] Starting up docker-dns-updater daemon")

	state := &state{}

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
	state.Config = configuration
	log.Printf("[INFO] Using configuration: %s", state.Config)

	// Connect to docker daemon
	log.Println("[INFO] Connecting to docker")
	dockerClient, err := getDockerClient()
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize docker client: %s", err)
	}
	state.DockerClient = dockerClient
	log.Println("[INFO] Connected to docker")

	// Create the DNSProvider
	log.Printf("[INFO] Connecting to DNS Provider: %s", state.Config.Provider)
	provider, err := getDNSProvider(state.Config)
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect with DNS Provider %s", err)
	}
	state.Provider = provider
	log.Printf("[INFO] Connected to DNS Provider: %s", state.Config.Provider)

	// Create the store
	log.Printf("[INFO] Connecting to Store: %s", state.Config.Store)
	db, err := getStore(state.Config)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create state store: %s", err)
	}
	state.Store = db
	defer state.Store.CleanUp()
	log.Printf("[INFO] Connected to Store: %s", state.Config.Store)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Fetch the current state
	syncDNSWithDocker(state)

	// Monitor for changes and update the current state
	monitorEvents(state, signalChan)
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

func syncDNSWithDocker(state *state) {
	// Assert that the we are in a valid state
	assertNotNil(state.DockerClient, "DockerClient not initialized")
	assertNotNil(state.Store, "Store not initialized")
	assertNotNil(state.Provider, "DNSProvider not initialized")
	assert(state.Config.DockerLabel != "", "Empty DockerLabel provided")

	args := filters.NewArgs()
	args.Add("label", state.Config.DockerLabel)
	args.Add("status", "running")
	containerList, err := state.DockerClient.ContainerList(context.Background(), dt.ContainerListOptions{
		Filters: args,
	})
	// TODO: propagate error out of this function
	if err != nil {
		log.Fatalf("[FATAL] Failed to list docker containers: %s", err)
	}

	mappingList := make([]*types.DNSMapping, len(containerList))
	for i, container := range containerList {
		ip, err := getIP(&container, state.Config.DNSContent)
		if err != nil {
			log.Printf("[Error] Failed to obtain IP address for container %s: %s", container.ID, err)
			continue
		}
		mappingList[i] = &types.DNSMapping{
			Name:        container.Labels[state.Config.DockerLabel],
			IP:          ip,
			ContainerID: container.ID,
		}
	}
	err = state.Store.ReplaceMappings(mappingList, state.Provider)
	if err != nil {
		log.Fatalf("[FATAL] Failed to replace mappings: %s", err)
	}
}

func monitorEvents(state *state, signalChan chan os.Signal) {
	// Assert that the we are in a valid state
	assertNotNil(state.DockerClient, "DockerClient not initialized")
	assertNotNil(state.Store, "Store not initialized")
	assertNotNil(state.Provider, "DNSProvider not initialized")
	assert(state.Config.DockerLabel != "", "Empty DockerLabel provided")

	args := filters.NewArgs()
	args.Add("scope", "swarm")
	args.Add("scope", "local")
	//args.Add("type", "service") // service is created and deleted, we should probably special case this through some config
	args.Add("type", "container")
	//args.Add("type", "config") // TODO: check what triggers these events
	args.Add("event", "start")
	args.Add("event", "die")
	//args.Add("event", "update") // Only services are updated
	args.Add("label", state.Config.DockerLabel)

	// TODO: also listen to network/connect and network/disconnect messages, as these might change the IP of a container

	eventsChan, errorChan := state.DockerClient.Events(context.Background(), dt.EventsOptions{
		Filters: args,
	})

	for {
		select {
		case event := <-eventsChan:
			log.Printf("[DEBUG] Received event: %s", event)
			container, err := getContainerByID(state.DockerClient, event.Actor.ID)
			if err != nil {
				log.Printf("[ERROR] Could not obtain container details: %s", err)
				continue
			}
			ip, err := getIP(container, state.Config.DNSContent)
			if err != nil {
				log.Printf("[ERROR] Could not obtain container IP: %s", err)
				continue
			}
			mapping := &types.DNSMapping{
				Name:        event.Actor.Attributes[state.Config.DockerLabel],
				IP:          ip,
				ContainerID: event.Actor.ID,
			}
			switch event.Action {
			case "start":
				log.Printf("Check in memory store and create DNS if necessary")
				err = state.Store.InsertMapping(mapping, state.Provider.AddHostnameMapping)
				if err != nil {
					// TODO: just propagate the error out of this function and handle it higher up
					log.Fatalf("[FATAL] Encountered an error when updating DNS: %s", err)
				}
			case "die":
				log.Printf("Check in memory store and remove DNS if necessary")
				err := state.Store.RemoveMapping(mapping, state.Provider.RemoveHostnameMapping)
				if err != nil {
					// TODO: just propagate the error out of this function and handle it higher up
					log.Fatalf("[FATAL] Encountered an error when updating DNS: %s", err)
				}
			default:
				log.Printf("[WARN] Unsupported event")
			}
		case err := <-errorChan:
			log.Printf("[FATAL] Received a docker error: %s", err)
			return
		case sig := <-signalChan:
			log.Printf("[INFO] Received signal to terminate: %s", sig)
			return
		}
	}
}

// getIP returns an IP address for a given container. How the IP is determined is driven by mode:
//   * If mode is `container`: the IP address of the container in the first network is returned
//   * If mode is an IP address: that IP address is parsed and returned
func getIP(container *dt.Container, mode string) (net.IP, error) {
	switch mode {
	case "container":
		// TODO: look at a docker label for the network to use (return first if not set)
		for name, network := range container.NetworkSettings.Networks {
			log.Printf("[DEBUG] (network: %s, ip: %s)", name, network.IPAddress)
			if network.IPAddress != "" {
				return net.ParseIP(network.IPAddress), nil
			}
		}
		return nil, errors.New("container has no internal IP addresses")
	default:
		return net.ParseIP(mode), nil
	}
}

func getContainerByID(client *docker.Client, ID string) (*dt.Container, error) {
	args := filters.NewArgs()
	args.Add("id", ID)
	containers, err := client.ContainerList(context.Background(), dt.ContainerListOptions{
		Filters: args,
	})
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return nil, fmt.Errorf("no container with ID %s could be found", ID)
	}
	return &containers[0], nil
}

func assert(expr bool, msg string) {
	if !expr {
		log.Fatalf("Assertion Failed: %s", msg)
	}
}

func assertNotNil(value interface{}, msg string) {
	if value == nil {
		log.Fatalf("Not nil assertion failed: %s", msg)
	}
}
