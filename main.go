package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/wdullaer/docker-dns-updater/dns"
	"github.com/wdullaer/docker-dns-updater/stringslice"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	docker "github.com/docker/docker/client"
	memdb "github.com/hashicorp/go-memdb"
)

var (
	provider      = flag.String("provider", os.Getenv("PROVIDER"), "The DNS provider to register the domain names with")
	accountName   = flag.String("account-name", os.Getenv("ACCOUNT_NAME"), "The account-name (or equivalent) to be used for authenticating with the DNS provider")
	accountSecret = flag.String("account-secret", os.Getenv("ACCOUNT_SECRET"), "The account-secret (or equivalent) to be used for authenticating with the DNS provider")
	dnsContent    = flag.String("dns-content", os.Getenv("DNS_CONTENT"), "The IP address to be added to the DNS content (default: `container`, oneOf: [`container`, `<ipv4>`]")
	dockerLabel   = flag.String("docker-label", os.Getenv("DOCKER_LABEL"), "The docker label that contains the domain name")
)

type state struct {
	Config       *config
	Provider     dns.DNSProvider
	DockerClient *docker.Client
	Db           *memdb.MemDB
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

	// Create the in memory database
	db, err := initializeDatabase()
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize in memory database: %s", err)
	}
	state.Db = db

	// Fetch the current state
	syncDNSWithDocker(state)

	// Monitor for changes and update the current state
	monitorEvents(state)
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

func initializeDatabase() (*memdb.MemDB, error) {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"dns-label": &memdb.TableSchema{
				Name: "dns-label",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.CompoundIndex{Indexes: []memdb.Indexer{&memdb.StringFieldIndex{Field: "Name"}, &memdb.StringFieldIndex{Field: "IP"}}},
					},
					"containerid": &memdb.IndexSchema{
						Name:    "containerid",
						Unique:  true,
						Indexer: &memdb.StringSliceFieldIndex{Field: "ContainerID"},
					},
				},
			},
		},
	}

	return memdb.NewMemDB(schema)
}

func syncDNSWithDocker(state *state) {
	// Assert that the we are in a valid state
	assertNotNil(state.DockerClient, "DockerClient not initialized")
	assertNotNil(state.Db, "Db not initialized")
	assertNotNil(state.Provider, "DNSProvider not initialized")
	assert(state.Config.DockerLabel != "", "Empty DockerLabel provided")

	args := filters.NewArgs()
	args.Add("label", state.Config.DockerLabel)
	args.Add("status", "running")
	containerList, err := state.DockerClient.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: args,
	})
	// TODO: propagate error out of this function
	if err != nil {
		log.Fatalf("[FATAL] Failed to list docker containers: %s", err)
	}

	if len(containerList) != 0 {
		txn := state.Db.Txn(true)
		// TODO: improve this by only removing entries that are not present in the container list
		txn.DeleteAll("dns-label", "id")
		for i := range containerList {
			container := containerList[i]
			name := container.Labels[state.Config.DockerLabel]
			ip, err := getIP(&container, state.Config.DNSContent)
			if err != nil {
				log.Printf("[ERROR] Failed to obtain IP address: %s", err)
				continue
			}
			containerID := container.ID

			err = insertRecord(txn, name, ip, containerID, state.Provider)
			if err != nil {
				txn.Abort()
				// TODO: just propagate the error out of this function and handle it higher up
				log.Fatalf("[FATAL] Encountered an error when updating DNS: %s", err)
			}
		}
		txn.Commit()
	}
}

func monitorEvents(state *state) {
	// Assert that the we are in a valid state
	assertNotNil(state.DockerClient, "DockerClient not initialized")
	assertNotNil(state.Db, "Db not initialized")
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

	eventsChan, errorChan := state.DockerClient.Events(context.Background(), types.EventsOptions{
		Filters: args,
	})

	for {
		select {
		case event := <-eventsChan:
			log.Printf("[DEBUG] Received event: %s", event)
			name := event.Actor.Attributes[state.Config.DockerLabel]
			containerID := event.Actor.ID
			switch event.Action {
			case "start":
				container, err := getContainerByID(state.DockerClient, containerID)
				if err != nil {
					log.Printf("[ERROR] Could not obtain container details: %s", err)
					continue
				}
				ip, err := getIP(container, state.Config.DNSContent)
				if err != nil {
					log.Printf("[ERROR] Could not obtain container IP: %s", err)
					continue
				}
				log.Printf("Check in memory store and create DNS if necessary")
				txn := state.Db.Txn(true)
				err = insertRecord(txn, name, ip, containerID, state.Provider)
				if err != nil {
					txn.Abort()
					// TODO: just propagate the error out of this function and handle it higher up
					log.Fatalf("[FATAL] Encountered an error when updating DNS: %s", err)
				}
				txn.Commit()
			case "die":
				log.Printf("Check in memory store and remove DNS if necessary")
				txn := state.Db.Txn(true)
				err := removeRecord(txn, containerID, state.Provider)
				if err != nil {
					txn.Abort()
					// TODO: just propagate the error out of this function and handle it higher up
					log.Fatalf("[FATAL] Encountered an error when updating DNS: %s", err)
				}
				txn.Commit()
			default:
				log.Printf("[WARN] Unsupported event")
			}
		case err := <-errorChan:
			log.Fatalf("[FATAL] Received a docker error: %s", err)
			// TODO: cleanup
		}
	}
}

func insertRecord(txn *memdb.Txn, name string, ip string, containerID string, provider dns.DNSProvider) error {
	rawRecord, err := txn.First("dns-label", "id", name, ip)
	if err != nil {
		return err
	}
	if rawRecord == nil {
		log.Printf("[INFO] Insert record into DNS")
		if err = provider.AddHostnameMapping(name, ip); err != nil {
			return err
		}
		log.Print("[DEBUG] No record found in in-memory store: creating it")
		if err = txn.Insert("dns-label", &DNSLabel{
			Name:        name,
			IP:          ip,
			ContainerID: []string{containerID},
		}); err != nil {
			return err
		}
		return nil
	}
	record := rawRecord.(*DNSLabel)
	log.Printf("[DEBUG] Record found in in-memory store: %s", record)
	if !stringslice.Contains(record.ContainerID, containerID) {
		// No need to update DNS, the record should already exist
		log.Print("[DEBUG] Record does not contain containerID, adding it")
		if err = txn.Delete("dns-label", record); err != nil {
			return err
		}
		record.ContainerID = append(record.ContainerID, containerID)
		if err = txn.Insert("dns-label", record); err != nil {
			return err
		}
	}
	return nil
}

func updateRecord(txn *memdb.Txn, name string, ip string, containerID string, provider dns.DNSProvider) error {
	rawRecord, err := txn.First("dns-label", "containerid", containerID)
	if err != nil {
		return err
	}
	if rawRecord == nil {
		return insertRecord(txn, name, ip, containerID, provider)
	}
	record := rawRecord.(*DNSLabel)
	if record.Name == name && record.IP == ip {
		return nil
	}
	err = removeRecord(txn, containerID, provider)
	if err != nil {
		return err
	}
	return insertRecord(txn, name, ip, containerID, provider)
}

func removeRecord(txn *memdb.Txn, containerID string, provider dns.DNSProvider) error {
	rawRecord, err := txn.First("dns-label", "containerid", containerID)
	if err != nil {
		return err
	}
	if rawRecord == nil {
		log.Printf("[WARN] Trying to remove non-existing DNS-container mapping. (containerID: %s)", containerID)
		return nil
	}

	if err = txn.Delete("dns-label", rawRecord); err != nil {
		return err
	}

	record := rawRecord.(*DNSLabel)
	record.ContainerID = stringslice.RemoveFirst(record.ContainerID, containerID)

	if len(record.ContainerID) == 0 {
		// No more containers mapped to this (name, ip) tuple, it can be removed from DNS
		log.Printf("[INFO] Removing item from DNS")
		if err = provider.RemoveHostnameMapping(record.Name, record.IP); err != nil {
			return err
		}
	} else {
		log.Printf("[DEBUG] Record still has mappings, updating it")
		// Still containers mapped to this (name, ip) tuple, insert the updated record into the db
		if err = txn.Insert("dns-label", record); err != nil {
			return err
		}
	}

	return nil
}

func getIP(container *types.Container, mode string) (string, error) {
	switch mode {
	case "container":
		// TODO: look at a docker label for the network to use (return first if not set)
		for name, network := range container.NetworkSettings.Networks {
			log.Printf("[DEBUG] (network: %s, ip: %s)", name, network.IPAddress)
			if network.IPAddress != "" {
				return network.IPAddress, nil
			}
		}
		return "", errors.New("container has no internal IP addresses")
	default:
		return mode, nil
	}
}

func getContainerByID(client *docker.Client, ID string) (*types.Container, error) {
	args := filters.NewArgs()
	args.Add("id", ID)
	containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{
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

type DNSLabel struct {
	Name        string
	IP          string
	ContainerID []string
}
