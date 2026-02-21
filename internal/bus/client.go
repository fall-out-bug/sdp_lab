package bus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// DefaultReconnectWait is the wait between reconnect attempts.
const DefaultReconnectWait = 2 * time.Second

// DefaultMaxReconnects is the max reconnect attempts (0 = unlimited).
const DefaultMaxReconnects = 0

// Client manages NATS connection with reconnect and JetStream.
type Client struct {
	url            string
	nc             *nats.Conn
	js             nats.JetStreamContext
	mu             sync.RWMutex
	reconnectWait  time.Duration
	maxReconnects  int
	connectTimeout time.Duration
}

// ClientOption configures Client.
type ClientOption func(*Client)

// WithReconnectWait sets wait between reconnects.
func WithReconnectWait(d time.Duration) ClientOption {
	return func(c *Client) {
		c.reconnectWait = d
	}
}

// WithMaxReconnects sets max reconnect attempts (0 = unlimited).
func WithMaxReconnects(n int) ClientOption {
	return func(c *Client) {
		c.maxReconnects = n
	}
}

// WithConnectTimeout sets initial connection timeout.
func WithConnectTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.connectTimeout = d
	}
}

// NewClient creates a NATS client. Call Connect() to establish connection.
func NewClient(url string, opts ...ClientOption) *Client {
	c := &Client{
		url:            url,
		reconnectWait:  DefaultReconnectWait,
		maxReconnects:  DefaultMaxReconnects,
		connectTimeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Connect establishes NATS connection and initializes JetStream.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.nc != nil && c.nc.IsConnected() {
		return nil
	}

	opts := []nats.Option{
		nats.Timeout(c.connectTimeout),
		nats.ReconnectWait(c.reconnectWait),
		nats.MaxReconnects(c.maxReconnects),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				slog.Warn("nats disconnected", "err", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("nats reconnected", "url", nc.ConnectedUrl())
		}),
	}

	nc, err := nats.Connect(c.url, opts...)
	if err != nil {
		return fmt.Errorf("nats connect: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return fmt.Errorf("nats jetstream: %w", err)
	}

	c.nc = nc
	c.js = js
	return nil
}

// Conn returns the underlying NATS connection. Panics if not connected.
func (c *Client) Conn() *nats.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.nc == nil {
		panic("bus: not connected")
	}
	return c.nc
}

// JetStream returns the JetStream context. Panics if not connected.
func (c *Client) JetStream() nats.JetStreamContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.js == nil {
		panic("bus: not connected")
	}
	return c.js
}

// IsConnected returns true if connected to NATS.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.nc != nil && c.nc.IsConnected()
}

// Close closes the NATS connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.nc != nil {
		c.nc.Close()
		c.nc = nil
		c.js = nil
	}
}
