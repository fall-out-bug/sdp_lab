package bus

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("nats://localhost:4222")
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestNewClient_options(t *testing.T) {
	c := NewClient("nats://localhost",
		WithReconnectWait(5*time.Second),
		WithMaxReconnects(3),
		WithConnectTimeout(1*time.Second),
	)
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestNewClient_Connect_noServer(t *testing.T) {
	c := NewClient("nats://127.0.0.1:19999", WithConnectTimeout(100*time.Millisecond))
	ctx := context.Background()
	err := c.Connect(ctx)
	if err == nil {
		t.Error("expected Connect to fail when no server")
	}
}
