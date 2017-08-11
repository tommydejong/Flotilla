package kafka

import (
	"fmt"
	"log"
	"os/exec"
	"time"
)

const (
	zookeeper     = "zookeeper"
<<<<<<< HEAD
	zookeeperCmd  = "docker run -d -p %s:%s %s"
=======
	zookeeperCmd  = "docker run -d --name zookeeper -p %s:%s %s"
>>>>>>> 44e1a4f21f68eeb20caa9e19c4a4ac1191f7201d
	zookeeperPort = "2181"
	kafka         = "ches/kafka"
	kafkaPort     = "9092"
	jmxPort       = "7203"
	kafkaCmd	  = "docker run -d --link zookeeper:zookeeper --hostname %s --publish %s:%s --publish %s:%s --env KAFKA_ADVERTISED_HOST_NAME=%s --env ZOOKEEPER_IP=%s %s"
)

// Broker implements the broker interface for Kafka.
type Broker struct {
	kafkaContainerID     string
	zookeeperContainerID string
}

// Start will start the message broker and prepare it for testing.
func (k *Broker) Start(host, port string) (interface{}, error) {
	if port == zookeeperPort || port == jmxPort {
		return nil, fmt.Errorf("Port %s is reserved", port)
	}

	cmd := fmt.Sprintf(zookeeperCmd, zookeeperPort, zookeeperPort, zookeeper)
	log.Printf("zookeeperCmd: %s", cmd)
	zkContainerID, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		log.Printf("Failed to start container %s: %s", zookeeper, err.Error())
		return "", err
	}
	log.Printf("Started container %s: %s", zookeeper, zkContainerID)

	localhostIP := "127.0.0.1"
	cmd = fmt.Sprintf(kafkaCmd, host, kafkaPort, kafkaPort, jmxPort, jmxPort, localhostIP, localhostIP, kafka)
	log.Printf("kafkaCmd: %s", cmd)
	kafkaContainerID, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		log.Printf("Failed to start container %s: %s", kafka, err.Error())
		k.Stop()
		return "", err
	}

	log.Printf("Started container %s: %s", kafka, kafkaContainerID)
	k.kafkaContainerID = string(kafkaContainerID)
	k.zookeeperContainerID = string(zkContainerID)

	// NOTE: Leader election can take a while. For now, just sleep to try to
	// ensure the cluster is ready. Is there a way to avoid this or make it
	// better?
	time.Sleep(time.Second * 15)
	log.Printf("Leader election complete")

	return string(kafkaContainerID), nil
}

// Stop will stop the message broker.
func (k *Broker) Stop() (interface{}, error) {
	_, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker kill %s", k.zookeeperContainerID)).Output()
	if err != nil {
		log.Printf("Failed to stop container %s: %s", zookeeper, err)
	} else {
		log.Printf("Stopped container %s: %s", zookeeper, k.zookeeperContainerID)
	}

	kafkaContainerID, e := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker kill %s", k.kafkaContainerID)).Output()
	if e != nil {
		log.Printf("err: %s", err)
		log.Printf("Failed to stop container %s", kafka)
		err = e
	} else {
		log.Printf("Stopped container %s: %s", kafka, k.kafkaContainerID)
	}

	return string(kafkaContainerID), err
}
