# Docker DNS Updater
`docker-dns-updater` is a utility that will create and update DNS A records in a DNS provider based on label data from running containers.

It is meant to be used in parallel with a reverse proxy like [traefik](https://traefik.io) or [caddy](https://github.com/lucaslorentz/caddy-docker-proxy): you can expose multiple services with a unique name on the same host. While you can also use this tool for cloud native service discovery, there are more battle tested tools out there (such as [consul](https://consul.io))

Both the DNS provider and the internal state store are pluggable. This allows the tool to be used in a variety of scenario's.

Currently, the following state stores are supported:
* Ephemeral in memory database
* Persistent embedded boltdb

Currently, the following DNS providers are supported: 
* [Cloudflare](https://www.cloudflare.com/)
* Dryrun: not an actual provider, but prints all changes to the console. Useful to test out if the configuration is behaving as expected

## Limitations
The following limitations apply, not because they are out of scope per se, but because I didn't need them for my use case:
* Only IP v4 is supported
* Only A records can be created
* Host IP must be manually specified or the first internal container IP  
  Obtaining the host IP would require running on host or mounting host network on the container and even then a lot of config is required to find the correct one. I think it's just easier for the user to do this up front for now.

## Architecture
The application relies on 3 core entities:

```
┌──────────────────────────────────────┐
│                                      │
│                                      │
│                Docker                │
│                                      │
│                                      │
└──────────────────────────────────────┘
                    │                   
                    │ Container Events  
                    ▼                   
┌──────────────────────────────────────┐
│                                      │
│                Store                 │
│      A Record -> []ContainerID       │
│                                      │
│                                      │
└──────────────────────────────────────┘
                    │ New A Record      
                    │ Removed A Record  
                    ▼                   
┌──────────────────────────────────────┐
│                                      │
│                                      │
│             DNSProvider              │
│                                      │
│                                      │
└──────────────────────────────────────┘
```

* **Docker**  
  The docker daemon supplies the data with which the DNS provider is configured. At startup the current state of the daemon is inquired and processed. Afterwards incremental changes are processed by listening to docker container events.
* **Store**  
  The store keeps a mapping of A records to containerIDs. Since an A record can be required by multiple containers, we cannot just blindly update the DNSProvider based on the docker events and need to keep this piece of state
* **DNSProvider**  
  The DNSProvider abstracts the interaction with the API of the service provider. It provides methods to insert and remove A records

Currently the Store is responsible for interacting with the DNSProvider. The current store implementations will try to minimize the amount of API calls made to the DNSProvider. The DNSProvider interactions are also executed in a transaction to ensure the internal state is consistent with the remote state at the service provider.

## Build

### Native
```bash
go build
```

### Docker
```bash
docker build .
```

## TODO / Improvement Idea's
* [ ] Look up the network which IP address should be taken from a docker label
* [ ] Look into [viper config library](https://github.com/spf13/viper)
* [ ] Move to a better designed logging library like https://github.com/hashicorp/go-hclog
* [ ] Look into implementing DNS providers via a plugin using the [hashicorp plugin rpc](https://github.com/hashicorp/go-plugin)
