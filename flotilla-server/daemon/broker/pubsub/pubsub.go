package pubsub

import (
	"fmt"
	"log"
	"strings"
	"sync/atomic"

	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"

	"github.com/jack0/Flotilla/flotilla-server/daemon/broker"
)

const (
	stopped = 1

	// batchSize is the number of messages we try to publish and consume at a
	// time to increase throughput. TODO: this might need tweaking.
	batchSize = 100
)

// Peer implements the peer interface for Google Cloud Pub/Sub.
type Peer struct {
	client       *pubsub.Client
	context      context.Context
	subscription string
	messages     chan []byte
	stopped      int32
	acks         chan []string
	ackDone      chan bool
	send         chan []byte
	errors       chan error
	done         chan bool
	flush        chan bool
}

// NewPeer creates and returns a new Peer for communicating with Google Cloud
// Pub/Sub.
func NewPeer(projectID string) (*Peer, error) {
	ctx := context.Background()
	client, err := newClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &Peer{
		client:   client,
		context:  ctx,
		messages: make(chan []byte, 10000),
		acks:     make(chan []string, 100),
		ackDone:  make(chan bool, 1),
		send:     make(chan []byte),
		errors:   make(chan error, 1),
		done:     make(chan bool),
		flush:    make(chan bool),
	}, nil
}

// Subscribe prepares the peer to consume messages.
func (c *Peer) Subscribe() error {
	// Subscription names must start with a lowercase letter, end with a
	// lowercase letter or number, and contain only lowercase letters, numbers,
	// dashes, underscores or periods.
	c.subscription = strings.ToLower(fmt.Sprintf("x%sx", broker.GenerateName()))
	sub := c.client.Subscription(c.subscription)
	exists, err := sub.Exists(c.context)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("Subscription %s already exists", c.subscription)
	}

	topic := c.client.Topic(topicName)

	if _, err := c.client.CreateSubscription(c.context, c.subscription, topic, 0, nil); err != nil {
		return err
	}

	go c.ack()

	go func() {
		// TODO: Can we avoid using atomic flag?
		for atomic.LoadInt32(&c.stopped) != stopped {
			messages, err := sub.Pull(c.context, pubsub.MaxPrefetch(batchSize))
			if err != nil {
				// Timed out.
				continue
			}

			ids := make([]string, 0)
			i := 0
			for message, err := messages.Next(); err != pubsub.Done; message, err = messages.Next() {
				ids[i] = message.ID
				c.messages <- message.Data
				message.Done(true)
				i++
			}
			c.acks <- ids
		}
	}()
	return nil
}

// Recv returns a single message consumed by the peer. Subscribe must be called
// before this. It returns an error if the receive failed.
func (c *Peer) Recv() ([]byte, error) {
	return <-c.messages, nil
}

// Send returns a channel on which messages can be sent for publishing.
func (c *Peer) Send() chan<- []byte {
	return c.send
}

// Errors returns the channel on which the peer sends publish errors.
func (c *Peer) Errors() <-chan error {
	return c.errors
}

// Done signals to the peer that message publishing has completed.
func (c *Peer) Done() {
	c.done <- true
	<-c.flush
}

// Setup prepares the peer for testing.
func (c *Peer) Setup() {
	buffer := make([]*pubsub.Message, batchSize)
	go func() {
		i := 0
		for {
			topic := c.client.Topic(topicName)
			select {
			case msg := <-c.send:
				buffer[i] = &pubsub.Message{Data: msg}
				i++
				if i == batchSize {
					if _, err := topic.Publish(c.context, buffer[i]); err != nil {
						c.errors <- err
					}
					i = 0
				}
			case <-c.done:
				if i > 0 {

					if _, err := topic.Publish(c.context, buffer[0:i]...); err != nil {
						c.errors <- err
					}
				}
				c.flush <- true
				return
			}
		}
	}()
}

// Teardown performs any cleanup logic that needs to be performed after the
// test is complete.
func (c *Peer) Teardown() {
	atomic.StoreInt32(&c.stopped, stopped)
	c.ackDone <- true
	sub := c.client.Subscription(c.subscription)
	sub.Delete(c.context)
}

func (c *Peer) ack() {
	for {
		select {
		case ids := <-c.acks:
			if len(ids) > 0 {
				if err := pubsub.Done; err != nil {
					log.Println("Failed to ack messages")
				}
			}
		case <-c.ackDone:
			return
		}
	}
}
