package daemon

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sdp_dev/internal/trace"
	"sdp_dev/internal/trace/bead"
)

const (
	// DefaultSocketTimeout is the default timeout for socket operations
	DefaultSocketTimeout = 5 * time.Second
	// MaxMessageSize is the maximum size of a message (10 MB)
	MaxMessageSize = 10 * 1024 * 1024
)

// SpanRegistry holds active spans keyed by (trace_id, tool_call_id)
type SpanRegistry struct {
	mu    sync.RWMutex
	spans map[string]*trace.SpanHandle // key: "trace_id:tool_call_id"
}

// NewSpanRegistry creates a new span registry
func NewSpanRegistry() *SpanRegistry {
	return &SpanRegistry{
		spans: make(map[string]*trace.SpanHandle),
	}
}

// Register registers a span in the registry
func (r *SpanRegistry) Register(traceID, toolCallID string, handle *trace.SpanHandle) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", traceID, toolCallID)
	r.spans[key] = handle
	return nil
}

// Lookup retrieves a span from the registry
func (r *SpanRegistry) Lookup(traceID, toolCallID string) (*trace.SpanHandle, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", traceID, toolCallID)
	handle, ok := r.spans[key]
	return handle, ok
}

// Remove removes a span from the registry
func (r *SpanRegistry) Remove(traceID, toolCallID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", traceID, toolCallID)
	delete(r.spans, key)
}

// Daemon is the session-lifetime trace daemon
type Daemon struct {
	socketPath   string
	tracesDir    string
	sessionID    string
	registry     *SpanRegistry
	traceGen     *bead.TraceIDGenerator
	spanGen      *bead.SpanIDGenerator
	currentFile  *os.File
	currentDate  string
	schema       *trace.Schema
	consentLevel trace.ConsentLevel
	attrFilter   *trace.AttributeFilter
	mu           sync.Mutex
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

// Config holds daemon configuration
type Config struct {
	SocketPath   string
	TracesDir    string
	SessionID    string
	Schema       *trace.Schema
	ConsentLevel trace.ConsentLevel
}

// NewDaemon creates a new trace daemon
func NewDaemon(config *Config) *Daemon {
	return &Daemon{
		socketPath:   config.SocketPath,
		tracesDir:    config.TracesDir,
		sessionID:    config.SessionID,
		registry:     NewSpanRegistry(),
		traceGen:     bead.NewTraceIDGenerator(),
		spanGen:      bead.NewSpanIDGenerator(),
		schema:       config.Schema,
		consentLevel: config.ConsentLevel,
		attrFilter:   trace.NewAttributeFilter(*config.Schema, config.ConsentLevel),
		shutdownChan: make(chan struct{}),
	}
}

// Start starts the daemon
func (d *Daemon) Start() error {
	// Ensure socket directory exists
	socketDir := filepath.Dir(d.socketPath)
	if err := os.MkdirAll(socketDir, 0700); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket if present
	os.Remove(d.socketPath)

	// Create Unix domain socket
	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}

	// Set socket permissions to owner-only
	if err := os.Chmod(d.socketPath, 0600); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Ensure traces directory exists
	if err := os.MkdirAll(d.tracesDir, 0755); err != nil {
		listener.Close()
		return fmt.Errorf("failed to create traces directory: %w", err)
	}

	// Open current trace file
	if err := d.openTraceFile(); err != nil {
		listener.Close()
		return fmt.Errorf("failed to open trace file: %w", err)
	}

	d.wg.Add(1)
	go d.run(listener)

	return nil
}

// run is the main daemon loop
func (d *Daemon) run(listener net.Listener) {
	defer d.wg.Done()
	defer listener.Close()
	defer func() {
		_ = d.closeTraceFile()
	}()

	for {
		select {
		case <-d.shutdownChan:
			return
		default:
			// Set accept deadline to allow shutdown checking
			unixListener, ok := listener.(*net.UnixListener)
			if !ok {
				return
			}
			if err := unixListener.SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
				continue
			}

			conn, err := listener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Timeout is normal, check shutdown
				}
				continue // Log error in production
			}

			d.wg.Add(1)
			go d.handleConnection(conn)
		}
	}
}

// handleConnection handles a single client connection
func (d *Daemon) handleConnection(conn net.Conn) {
	defer d.wg.Done()
	defer conn.Close()

	// Set read deadline
	if err := conn.SetDeadline(time.Now().Add(DefaultSocketTimeout)); err != nil {
		return
	}

	// Read length-prefixed JSON message
	reader := bufio.NewReader(conn)

	// Read message length (4 bytes)
	var length uint32
	if err := binaryRead(reader, &length); err != nil {
		return
	}

	if length > MaxMessageSize {
		// Send error response
		d.sendError(conn, fmt.Sprintf("message too large: %d bytes", length))
		return
	}

	// Read message body
	data := make([]byte, length)
	if _, err := io.ReadFull(reader, data); err != nil {
		return
	}

	// Parse request
	var req trace.DaemonRequest
	if err := json.Unmarshal(data, &req); err != nil {
		d.sendError(conn, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	// Handle request
	resp := d.handleRequest(&req)

	// Send response
	d.sendResponse(conn, resp)
}

// handleRequest processes a daemon request
func (d *Daemon) handleRequest(req *trace.DaemonRequest) *trace.DaemonResponse {
	switch req.Command {
	case "span-start":
		return d.handleSpanStart(req)
	case "span-end":
		return d.handleSpanEnd(req)
	case "event":
		return d.handleEvent(req)
	case "shutdown":
		return &trace.DaemonResponse{Success: true}
	default:
		return &trace.DaemonResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown command: %s", req.Command),
		}
	}
}

// handleSpanStart handles a span-start request
func (d *Daemon) handleSpanStart(req *trace.DaemonRequest) *trace.DaemonResponse {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Generate IDs
	traceID := d.traceGen.Generate()
	spanID := d.spanGen.Generate()

	// Create span
	span := &trace.Span{
		TraceID:    traceID,
		SpanID:     spanID,
		SpanKind:   trace.SpanKind(req.Attributes["span_kind"]),
		StartTime:  time.Now(),
		Attributes: req.Attributes,
		Status:     trace.SpanStatusOK,
	}

	// Filter attributes based on consent level
	spanKind := string(span.SpanKind)
	span.Attributes = d.attrFilter.Filter(spanKind, span.Attributes)

	// Create handle
	handle := &trace.SpanHandle{
		Span:      span,
		TraceID:   traceID,
		StartTime: time.Now(),
	}

	// Register in span registry
	if toolCallID := req.ToolCallID; toolCallID != "" {
		if err := d.registry.Register(traceID, toolCallID, handle); err != nil {
			return &trace.DaemonResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to register span: %v", err),
			}
		}
		handle.ToolCallID = toolCallID
	}

	return &trace.DaemonResponse{
		Success: true,
		SpanID:  spanID,
		TraceID: traceID,
	}
}

// handleSpanEnd handles a span-end request
func (d *Daemon) handleSpanEnd(req *trace.DaemonRequest) *trace.DaemonResponse {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Lookup span by tool_call_id or span_id
	var handle *trace.SpanHandle
	var ok bool

	// Try to lookup by tool_call_id first (preferred for concurrent tool calls)
	if req.ToolCallID != "" {
		// Search all spans in registry to find matching tool_call_id
		// Since registry key is "trace_id:tool_call_id", we need to search
		d.registry.mu.RLock()
		for _, h := range d.registry.spans {
			// Key format is "trace_id:tool_call_id"
			if h.ToolCallID == req.ToolCallID {
				handle = h
				ok = true
				break
			}
		}
		d.registry.mu.RUnlock()
	}

	// If not found by tool_call_id, try by span_id
	if !ok && req.SpanID != "" {
		d.registry.mu.RLock()
		for _, h := range d.registry.spans {
			if h.Span.SpanID == req.SpanID {
				handle = h
				ok = true
				break
			}
		}
		d.registry.mu.RUnlock()
	}

	if !ok || handle == nil {
		return &trace.DaemonResponse{
			Success: false,
			Error:   "span not found",
		}
	}

	// Update span
	span := handle.Span
	now := time.Now()
	span.EndTime = now
	span.DurationMs = now.Sub(span.StartTime).Milliseconds()

	// Apply attributes from request
	for k, v := range req.Attributes {
		span.Attributes[k] = v
	}

	// Write to trace file
	if err := d.writeSpan(span); err != nil {
		return &trace.DaemonResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to write span: %v", err),
		}
	}

	// Remove from registry
	if handle.ToolCallID != "" {
		d.registry.Remove(handle.TraceID, handle.ToolCallID)
	}

	return &trace.DaemonResponse{Success: true}
}

// handleEvent handles an event request
func (d *Daemon) handleEvent(req *trace.DaemonRequest) *trace.DaemonResponse {
	d.mu.Lock()
	defer d.mu.Unlock()

	// For MVP, events are just logged
	// In production, would add to span's events list
	return &trace.DaemonResponse{Success: true}
}

// writeSpan writes a span to the trace file
func (d *Daemon) writeSpan(span *trace.Span) error {
	// Rotate if date changed
	if err := d.checkRotation(); err != nil {
		return err
	}

	// Serialize span to JSON
	data, err := json.Marshal(span)
	if err != nil {
		return fmt.Errorf("failed to marshal span: %w", err)
	}

	// Write to file with newline
	if _, err := d.currentFile.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write span: %w", err)
	}

	return nil
}

// openTraceFile opens the current day's trace file
func (d *Daemon) openTraceFile() error {
	d.currentDate = time.Now().Format("2006-01-02")
	dateDir := filepath.Join(d.tracesDir, d.currentDate)

	// Create date directory
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return fmt.Errorf("failed to create date directory: %w", err)
	}

	// Open file (plain JSONL, no compression for live writes)
	filePath := filepath.Join(dateDir, "spans.jsonl")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open trace file: %w", err)
	}

	d.currentFile = file
	return nil
}

// closeTraceFile closes the current trace file
func (d *Daemon) closeTraceFile() error {
	if d.currentFile != nil {
		if err := d.currentFile.Close(); err != nil {
			return err
		}
		d.currentFile = nil
	}
	return nil
}

// checkRotation checks if the trace file needs rotation
func (d *Daemon) checkRotation() error {
	newDate := time.Now().Format("2006-01-02")
	if newDate != d.currentDate {
		// Close current file
		if err := d.closeTraceFile(); err != nil {
			return err
		}

		// Compress previous day's file (if exists)
		if d.currentDate != "" {
			oldFile := filepath.Join(d.tracesDir, d.currentDate, "spans.jsonl")
			if _, err := os.Stat(oldFile); err == nil {
				// Compress with zstd -3
				// For MVP, just log; actual compression done by cron
			}
		}

		// Open new file
		return d.openTraceFile()
	}
	return nil
}

// Shutdown gracefully shuts down the daemon
func (d *Daemon) Shutdown() error {
	close(d.shutdownChan)
	d.wg.Wait()

	// Close and remove socket
	if err := d.closeTraceFile(); err != nil {
		return err
	}
	if err := os.Remove(d.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// sendResponse sends a response to the client
func (d *Daemon) sendResponse(conn net.Conn, resp *trace.DaemonResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}

	// Write length prefix
	length := uint32(len(data))
	if err := binaryWrite(conn, length); err != nil {
		return
	}

	// Write body
	_, _ = conn.Write(data)
}

// sendError sends an error response to the client
func (d *Daemon) sendError(conn net.Conn, errMsg string) {
	resp := &trace.DaemonResponse{
		Success: false,
		Error:   errMsg,
	}
	d.sendResponse(conn, resp)
}

// binaryRead reads a binary value in big-endian format
func binaryRead(r io.Reader, v interface{}) error {
	return binary.Read(r, binary.BigEndian, v)
}

// binaryWrite writes a binary value in big-endian format
func binaryWrite(w io.Writer, v interface{}) error {
	return binary.Write(w, binary.BigEndian, v)
}
