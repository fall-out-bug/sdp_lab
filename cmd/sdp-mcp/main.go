// sdp-mcp is the MCP (Model Context Protocol) server for SDP toolkit commands.
// It exposes scout, architect, metrics, spec, index, bootstrap, dispatch,
// and beads operations as MCP tools over stdio transport.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	sdpmcp "sdp_dev/internal/mcp"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// ---------------------------------------------------------------------------
// Minimal log-level filtering (SDP_LOG_LEVEL env var)
// ---------------------------------------------------------------------------

type logLevel int

const (
	levelDebug logLevel = iota
	levelInfo
	levelWarn
	levelError
)

var currentLevel logLevel

func init() {
	switch os.Getenv("SDP_LOG_LEVEL") {
	case "debug":
		currentLevel = levelDebug
	case "info":
		currentLevel = levelInfo
	case "error":
		currentLevel = levelError
	default:
		currentLevel = levelWarn
	}
}

func logDebug(format string, args ...interface{}) {
	if currentLevel <= levelDebug {
		log.Printf(format, args...)
	}
}

func logInfo(format string, args ...interface{}) {
	if currentLevel <= levelInfo {
		log.Printf(format, args...)
	}
}

func logWarn(format string, args ...interface{}) {
	if currentLevel <= levelWarn {
		log.Printf(format, args...)
	}
}

func logError(format string, args ...interface{}) {
	if currentLevel <= levelError {
		log.Printf(format, args...)
	}
}

func main() {
	repo := flag.String("repo", ".", "repository root path (default: current directory)")
	binary := flag.String("binary", "", "path to sdp CLI binary (default: lookup 'sdp' in PATH)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s %s\n", sdpmcp.ServerName, sdpmcp.ServerVersion)
		os.Exit(0)
	}

	srv := sdpmcp.NewServer(sdpmcp.ServerConfig{
		BinaryPath: *binary,
		RepoRoot:   *repo,
	})

	logInfo("%s %s starting (repo=%s, binary=%s)",
		sdpmcp.ServerName, sdpmcp.ServerVersion,
		*repo, func() string {
			if *binary != "" {
				return *binary
			}
			return "sdp (PATH)"
		}())

	if err := mcpserver.ServeStdio(srv.Inner()); err != nil {
		logError("server error: %v", err)
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
