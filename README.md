# Docker DNS Updater
`docker-dns-updater` is a utility that will create and update DNS A records in a DNS provider based on label data from running containers.

It is meant to be used in parallel with a reverse proxy like [traefik](https://traefik.io) or [caddy](https://github.com/lucaslorentz/caddy-docker-proxy): you can expose multiple services with a unique name on the same host. While you can also use this tool for cloud native service discovery, there are more battle tested tools out there (such as [consul](https://consul.io))

Currently, the following DNS providers are supported: 
* Cloudflare

## Limitations
The following limitations apply, not because they are out of scope per se, but because I didn't need them for my use case:
* Only IP v4 is supported
* Only A records can be created
* Host IP must be manually specified  
  Obtaining it would require running on host or mounting host network on the container and even then a lot of config is required to find the correct one. I think it's just easier for the user to do this up front for now.

## TODO
* [ ] Look into [viper config library](https://github.com/spf13/viper)
* [ ] Move to a better designed logging library like https://github.com/hashicorp/go-hclog
* [ ] Allow DNS providers to be implemented via a plugin using the [hashicorp plugin rpc](https://github.com/hashicorp/go-plugin)
* [ ] Make embedded db pluggable (eg use consul K/V so it can run in a cluster)
