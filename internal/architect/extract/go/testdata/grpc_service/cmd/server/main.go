package main

import (
	"github.com/example/grpcservice/internal/server"
	"google.golang.org/grpc"
)

func main() {
	s := grpc.NewServer()
	server.RegisterGreeterServer(s)
	// Start server
}
