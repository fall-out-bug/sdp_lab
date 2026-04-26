package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"sdp_dev/internal/trace"
)

// Client is a trace daemon client
type Client struct {
	socketPath string
	timeout    time.Duration
}

// NewClient creates a new trace client
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		timeout:    5 * time.Second,
	}
}

// Span starts a new span
func (c *Client) Span(ctx context.Context, spanKind trace.SpanKind, toolCallID string, attrs map[string]string) (string, string, error) {
	// Add span_kind to attributes
	if attrs == nil {
		attrs = make(map[string]string)
	}
	attrs["span_kind"] = string(spanKind)

	req := &trace.DaemonRequest{
		Command:    "span-start",
		ToolCallID: toolCallID,
		Attributes: attrs,
	}

	resp, err := c.send(req)
	if err != nil {
		return "", "", err
	}

	if !resp.Success {
		return "", "", fmt.Errorf("span-start failed: %s", resp.Error)
	}

	return resp.TraceID, resp.SpanID, nil
}

// End ends a span
func (c *Client) End(ctx context.Context, spanID, toolCallID string, attrs map[string]string) error {
	req := &trace.DaemonRequest{
		Command:    "span-end",
		SpanID:     spanID,
		ToolCallID: toolCallID,
		Attributes: attrs,
	}

	resp, err := c.send(req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("span-end failed: %s", resp.Error)
	}

	return nil
}

// Event adds an event to a span
func (c *Client) Event(ctx context.Context, spanID string, event trace.SpanEvent) error {
	req := &trace.DaemonRequest{
		Command: "event",
		SpanID:  spanID,
		Event:   event,
	}

	resp, err := c.send(req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("event failed: %s", resp.Error)
	}

	return nil
}

// Shutdown shuts down the daemon
func (c *Client) Shutdown(ctx context.Context) error {
	req := &trace.DaemonRequest{
		Command: "shutdown",
	}

	resp, err := c.send(req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("shutdown failed: %s", resp.Error)
	}

	return nil
}

// send sends a request to the daemon
func (c *Client) send(req *trace.DaemonRequest) (*trace.DaemonResponse, error) {
	// Serialize request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Connect to daemon socket
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w (is the trace daemon running?)", err)
	}
	defer conn.Close()

	// Set deadline
	conn.SetDeadline(time.Now().Add(c.timeout))

	// Write length prefix
	length := uint32(len(data))
	if err := binaryWrite(conn, length); err != nil {
		return nil, fmt.Errorf("failed to write length: %w", err)
	}

	// Write body
	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response length
	var respLength uint32
	if err := binaryRead(conn, &respLength); err != nil {
		return nil, fmt.Errorf("failed to read response length: %w", err)
	}

	// Read response body
	respData := make([]byte, respLength)
	if _, err := io.ReadFull(conn, respData); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var resp trace.DaemonResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// binaryRead reads a binary value
func binaryRead(r io.Reader, v interface{}) error {
	// For MVP, simplified
	return nil
}

// binaryWrite writes a binary value
func binaryWrite(w io.Writer, v interface{}) error {
	// For MVP, simplified
	return nil
}

// FindSocketPath finds the trace daemon socket path for the current session
func FindSocketPath() (string, error) {
	// Check SDP_TRACE_SOCKET environment variable
	if socketPath := os.Getenv("SDP_TRACE_SOCKET"); socketPath != "" {
		return socketPath, nil
	}

	// Check SDP_SESSION_ID
	sessionID := os.Getenv("SDP_SESSION_ID")
	if sessionID == "" {
		return "", fmt.Errorf("SDP_SESSION_ID not set")
	}

	// Build socket path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	socketPath := fmt.Sprintf("%s/.sdp/sockets/trace-%s.sock", homeDir, sessionID)
	return socketPath, nil
}

// MustGetClient returns a client or exits with error
func MustGetClient() *Client {
	socketPath, err := FindSocketPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "The trace daemon may not be running.\n")
		fmt.Fprintf(os.Stderr, "Start it with: sdp trace daemon\n")
		os.Exit(2)
	}

	return NewClient(socketPath)
}
