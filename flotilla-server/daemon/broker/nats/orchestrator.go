package nats

import (
	"fmt"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"
)

const (
	gnatsd       = "nats"
	internalPort = "4222"
)

// Broker implements the broker interface for NATS.
type Broker struct {
	containerID string
}

// Start will start the message broker and prepare it for testing.
func (n *Broker) Start(host, port string) (interface{}, error) {
	cli, err := client.NewEnvClient()
	ctx := context.Background()

	_ = cli.ContainerRemove(ctx, "nats-benchmark", types.ContainerRemoveOptions{Force: true})

	mappedPort := fmt.Sprintf("%s:%s/tcp", port, internalPort)

	portMap, portBindings, err := nat.ParsePortSpecs([]string{mappedPort, "6222:6222/tcp", "8222:8222/tcp"})

	container, err := cli.ContainerCreate(ctx, &container.Config{Image: "nats", ExposedPorts: portMap}, &container.HostConfig{PortBindings: portBindings}, nil, "nats-benchmark")
	if err != nil {
		log.Printf("Failed to create container %s: %s", gnatsd, err.Error())
		return "", err
	}
	n.containerID = container.ID

	err = cli.ContainerStart(ctx, container.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Printf("Failed to start container %s: %s", gnatsd, err.Error())
		return "", err
	}

	log.Printf("Started container %s: %s", gnatsd, container.ID)
	n.containerID = string(container.ID)
	return string(container.ID), nil
}

// Stop will stop the message broker.
func (n *Broker) Stop() (interface{}, error) {
	cli, err := client.NewEnvClient()
	ctx := context.Background()

	timeout := 30 * time.Second
	cli.ContainerStop(ctx, n.containerID, &timeout)
	if err != nil {
		log.Printf("Failed to stop container %s: %s", gnatsd, err.Error())
		return "", err
	}

	log.Printf("Stopped container %s: %s", gnatsd, n.containerID)
	err = cli.ContainerRemove(ctx, "nats-benchmark", types.ContainerRemoveOptions{Force: true})
	n.containerID = ""
	return string(n.containerID), nil
}
