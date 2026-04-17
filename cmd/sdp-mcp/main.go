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

	log.Printf("%s %s starting (repo=%s, binary=%s)",
		sdpmcp.ServerName, sdpmcp.ServerVersion,
		*repo, func() string {
			if *binary != "" {
				return *binary
			}
			return "sdp (PATH)"
		}())

	if err := mcpserver.ServeStdio(srv.Inner()); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
