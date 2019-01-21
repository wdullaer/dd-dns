package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	memdb "github.com/hashicorp/go-memdb"
)

var (
	provider      = flag.String("provider", os.Getenv("PROVIDER"), "The DNS provider to register the domain names with")
	accountName   = flag.String("account-name", os.Getenv("ACCOUNT_NAME"), "The account-name (or equivalent) to be used for authenticating with the DNS provider")
	accountSecret = flag.String("account-secret", os.Getenv("ACCOUNT_SECRET"), "The account-secret (or equivalent) to be used for authenticating with the DNS provider")
	dnsContent    = flag.String("dns-content", os.Getenv("DNS_CONTENT"), "The IP address to be added to the DNS content (default: `host`, oneOf: [`host`, `container`, `<ipv4>`]")
	dockerLabel   = flag.String("docker-label", os.Getenv("DOCKER_LABEL"), "The docker label that contains the domain name")
)

func main() {
	log.Println("[INFO] Starting up docker-dns-updater daemon")
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
	log.Printf("[INFO] Using configuration: %s", configuration)

	log.Println("[INFO] Connecting to docker")
	dockerClient, err := getDockerClient()
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize docker client: %s", err)
	}
	log.Println("[INFO] Connected to docker")

	log.Printf("[INFO] Connecting to DNS Provider: %s", "Cloudflare")
	provider, err := getDNSProvider()
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect with DNS Provider %s", err)
	}
	log.Printf("[INFO] Connected to DNS Provider: %s", "Cloudflare")

	db, err := initializeDatabase()
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize in memory database: %s", err)
	}

	monitorEvents(dockerClient, provider, db)
}

func getDockerClient() (*client.Client, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
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

func getDNSProvider() (DNSProvider, error) {
	// TODO: Add support for multiple providers
	return NewCloudflareProvider("TODO: Email", "TODO: Token")
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

func monitorEvents(dockerClient *client.Client, provider DNSProvider, db *memdb.MemDB) {
	// TODO: make the label we are looking for configurable
	dnsNameLabel := "caddy.address"
	args := filters.NewArgs()
	args.Add("scope", "swarm")
	args.Add("scope", "local")
	args.Add("type", "service") // service is created and deleted, we should probably special case this through some config
	args.Add("type", "container")
	args.Add("type", "config")
	args.Add("event", "start")
	args.Add("event", "die")
	args.Add("event", "update")
	args.Add("label", dnsNameLabel)

	eventsChan, errorChan := dockerClient.Events(context.Background(), types.EventsOptions{
		Filters: args,
	})

	for {
		select {
		case event := <-eventsChan:
			log.Printf("[DEBUG] Received event: %s", event)
			name := event.Actor.Attributes[dnsNameLabel]
			containerID := event.Actor.ID
			// TODO: make the target IP configurable, add an option so it's the internal container IP
			ip := "192.168.2.1"
			switch event.Action {
			case "start":
				log.Printf("Check in memory store and create DNS if necessary")
				txn := db.Txn(true)
				err := insertRecord(txn, name, ip, containerID, provider)
				if err != nil {
					txn.Abort()
					// TODO: just propagate the error out of this function and handle it higher up
					log.Fatalf("[FATAL] Encountered an error when updating DNS: %s", err)
				}
				txn.Commit()
			case "die":
				log.Printf("Check in memory store and remove DNS if necessary")
				txn := db.Txn(true)
				err := removeRecord(txn, containerID, provider)
				if err != nil {
					txn.Abort()
					// TODO: just propagate the error out of this function and handle it higher up
					log.Fatalf("[FATAL] Encountered an error when updating DNS: %s", err)
				}
				txn.Commit()
			case "update":
				// Labels cannot be updated at runtime for containers, only services, this is not necessary right now
				log.Print("Check the in memory store and update DNS if necessary")
				txn := db.Txn(true)
				err := updateRecord(txn, name, ip, containerID, provider)
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

func insertRecord(txn *memdb.Txn, name string, ip string, containerID string, provider DNSProvider) error {
	rawRecord, err := txn.First("dns-label", "id", name, ip)
	if err != nil {
		return err
	}
	if rawRecord == nil {
		// TODO: create record in DNS
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
	if !contains(record.ContainerID, containerID) {
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

func updateRecord(txn *memdb.Txn, name string, ip string, containerID string, provider DNSProvider) error {
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

func removeRecord(txn *memdb.Txn, containerID string, provider DNSProvider) error {
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
	record.ContainerID = remove(record.ContainerID, containerID)

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

func contains(col []string, item string) bool {
	for i := range col {
		if col[i] == item {
			return true
		}
	}
	return false
}

func remove(col []string, item string) []string {
	for i := range col {
		if col[i] == item {
			col[i] = col[len(col)-1]
			return col[:len(col)-1]
		}
	}
	return col
}

type DNSLabel struct {
	Name        string
	IP          string
	ContainerID []string
}
