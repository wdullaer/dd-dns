package main

import (
	"context"
	"errors"
	"fmt"
	"net"

	dt "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	docker "github.com/docker/docker/client"
	"github.com/wdullaer/dd-dns/types"
)

func syncDNSWithDocker(state *State) error {
	args := filters.NewArgs()
	args.Add("label", state.Config.DockerLabel)
	args.Add("status", "running")
	containerList, err := state.DockerClient.ContainerList(context.Background(), dt.ContainerListOptions{
		Filters: args,
	})
	if err != nil {
		return err
	}

	mappingList := make([]*types.DNSMapping, len(containerList))
	for i, container := range containerList {
		ip, err := getIP(&container, state.Config.DNSContent)
		if err != nil {
			state.Logger.Errorw("Failed to obtain IP address for container", "containerId", container.ID, "err", err)
			continue
		}
		mappingList[i] = &types.DNSMapping{
			Name:        container.Labels[state.Config.DockerLabel],
			IP:          ip,
			ContainerID: container.ID,
		}
	}

	state.Logger.Infow("Setting new mappings", "mappings", mappingList)
	return state.Store.ReplaceMappings(mappingList, state.Provider)
}

func processDockerEvent(event events.Message, state *State) error {
	container, err := getContainerByID(state.DockerClient, event.Actor.ID)
	if err != nil {
		state.Logger.Errorw("Could not obtain container details", "err", err)
		return nil
	}

	ip, err := getIP(container, state.Config.DNSContent)
	if err != nil {
		state.Logger.Errorw("Could not obtain container IP", "err", err)
		return nil
	}

	mapping := &types.DNSMapping{
		Name:        event.Actor.Attributes[state.Config.DockerLabel],
		IP:          ip,
		ContainerID: event.Actor.ID,
	}

	switch event.Action {
	case "start":
		state.Logger.Infow("Insert into store", "mapping", mapping)
		err = state.Store.InsertMapping(mapping, state.Provider.AddHostnameMapping)
		if err != nil {
			return err
		}
	case "die":
		state.Logger.Infow("Remove from store", "mapping", mapping)
		err := state.Store.RemoveMapping(mapping, state.Provider.RemoveHostnameMapping)
		if err != nil {
			return err
		}
	default:
		state.Logger.Warnw("Unsupported event", "event", event.Action)
	}

	return nil
}

func makeDockerChannels(client *docker.Client, config *config) (<-chan events.Message, <-chan error) {
	args := filters.NewArgs()
	args.Add("scope", "swarm")
	args.Add("scope", "local")
	//args.Add("type", "service") // service is created and deleted, we should probably special case this through some config
	args.Add("type", "container")
	//args.Add("type", "config") // TODO: check what triggers these events
	args.Add("event", "start")
	args.Add("event", "die")
	//args.Add("event", "update") // Only services are updated
	args.Add("label", config.DockerLabel)

	// TODO: also listen to network/connect and network/disconnect messages, as these might change the IP of a container

	return client.Events(context.Background(), dt.EventsOptions{
		Filters: args,
	})
}

// getContainerByID retrieves a Container Object. Returns an error if the container is not found
func getContainerByID(client *docker.Client, id string) (*dt.Container, error) {
	args := filters.NewArgs()
	args.Add("id", id)
	containers, err := client.ContainerList(context.Background(), dt.ContainerListOptions{
		Filters: args,
	})
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return nil, fmt.Errorf("no container with ID %s could be found", id)
	}
	return &containers[0], nil
}

// getIP returns an IP address for a given container. How the IP is determined is driven by mode:
//   * If mode is `container`: the IP address of the container in the first network is returned
//   * If mode is an IP address: that IP address is parsed and returned
func getIP(container *dt.Container, mode string) (net.IP, error) {
	switch mode {
	case "container":
		// TODO: look at a docker label for the network to use (return first if not set)
		for _, network := range container.NetworkSettings.Networks {
			if network.IPAddress != "" {
				return net.ParseIP(network.IPAddress), nil
			}
		}
		return nil, errors.New("container has no internal IP addresses")
	default:
		return net.ParseIP(mode), nil
	}
}
