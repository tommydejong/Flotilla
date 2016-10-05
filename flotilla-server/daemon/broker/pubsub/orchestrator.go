package pubsub

import (
	"errors"
	"log"

	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"
)

const topicName = "test"

// Broker implements the broker interface for Google Cloud Pub/Sub.
type Broker struct {
	ProjectID string
}

// Start will start the message broker and prepare it for testing.
func (c *Broker) Start(host, port string) (interface{}, error) {
	ctx := context.Background()
	client, err := newClient(ctx, c.ProjectID)
	if err != nil {
		return "", err
	}

	topic := client.Topic(topicName)
	exists, err := topic.Exists(ctx)
	if err != nil {
		log.Printf("Failed to check Cloud Pub/Sub topic: %s", err.Error())
		return "", err
	}

	if exists {
		if err := topic.Delete(ctx); err != nil {
			log.Printf("Failed to delete Cloud Pub/Sub topic: %s", err.Error())
			return "", err
		}
	}

	if _, err := client.CreateTopic(ctx, topicName); err != nil {
		log.Printf("Failed to create Cloud Pub/Sub topic: %s", err.Error())
		return "", err
	}

	log.Println("Created Cloud Pub/Sub topic")

	return "", nil
}

// Stop will stop the message broker.
func (c *Broker) Stop() (interface{}, error) {
	ctx := context.Background()
	client, err := newClient(ctx, c.ProjectID)
	if err != nil {
		return "", err
	}

	topic := client.Topic(topicName)
	if err := topic.Delete(ctx); err != nil {
		log.Printf("Failed to delete Cloud Pub/Sub topic: %s", err.Error())
		return "", err
	}

	log.Println("Deleted Cloud Pub/Sub topic")
	return "", err
}

func newClient(ctx context.Context, projectID string) (*pubsub.Client, error) {
	if ctx.Err != nil {
		return &pubsub.Client{}, errors.New("Invalid context.")
	}

	if projectID == "" {
		return &pubsub.Client{}, errors.New("project id not provided")
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return &pubsub.Client{}, err
	}

	return client, nil
}
