package rabbit

import (
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection manages an AMQP connection and a single channel.
// It implements lifecycle.Component.
type Connection struct {
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex
}

// New creates a Connection for the given AMQP URL.
// Call Start() to establish the connection.
func New(url string) *Connection {
	return &Connection{url: url}
}

// Name implements lifecycle.Component.
func (c *Connection) Name() string { return "rabbitmq" }

// Start implements lifecycle.Component.
// It dials the broker, opens a channel, and declares the topology.
func (c *Connection) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("amqp dial: %w", err)
	}
	c.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		conn.Close() //nolint:errcheck
		return fmt.Errorf("open channel: %w", err)
	}
	c.ch = ch

	if err := DeclareTopology(ch); err != nil {
		ch.Close()   //nolint:errcheck
		conn.Close() //nolint:errcheck
		return fmt.Errorf("declare topology: %w", err)
	}

	return nil
}

// Stop implements lifecycle.Component.
func (c *Connection) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ch != nil {
		c.ch.Close() //nolint:errcheck
	}
	if c.conn != nil {
		c.conn.Close() //nolint:errcheck
	}
	return nil
}

// Channel returns the AMQP channel.
// NOTE: amqp.Channel is NOT thread-safe; callers must use separate channels for
// concurrent publishing/consuming.
func (c *Connection) Channel() *amqp.Channel {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ch
}

// NewChannel opens and returns a fresh AMQP channel from the underlying connection.
// Callers own the returned channel and are responsible for closing it.
func (c *Connection) NewChannel() (*amqp.Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Channel()
}
