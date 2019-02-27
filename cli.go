package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	provider      = flag.String("provider", os.Getenv("PROVIDER"), "The DNS provider to register the domain names with (env: `PROVIDER`, default: `cloudflare`, oneOf: [`cloudflare`, `dryrun`])")
	accountName   = flag.String("account-name", os.Getenv("ACCOUNT_NAME"), "The account-name (or equivalent) to be used for authenticating with the DNS provider (env: `ACCOUNT_NAME`)")
	accountSecret = flag.String("account-secret", os.Getenv("ACCOUNT_SECRET"), "The account-secret (or equivalent) to be used for authenticating with the DNS provider (env: `ACCOUNT_SECRET`)")
	dnsContent    = flag.String("dns-content", os.Getenv("DNS_CONTENT"), "The IP address to be added to the DNS content (env: `DNS_CONTENT`, default: `container`, oneOf: [`container`, `<ipv4>`])")
	dockerLabel   = flag.String("docker-label", os.Getenv("DOCKER_LABEL"), "The docker label that contains the domain name (env: `DOCKER_LABEL`, default: `caddy.address`)")
	storeName     = flag.String("store", os.Getenv("STORE"), "The store implemenation that persists the internal state (env: `STORE`, default: `memory`, oneOf: [`memory`, `boltdb`])")
)

const usage = `Usage: %s [options]

  Watches the docker daemon configured in the current environment and maintains
  DNS records for running containers at a DNS provider.

  Options can be passed in as commandline flags or environment variables.
  Commandline flags take precedence over environment variables.
  
 Options:
 `

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
}
