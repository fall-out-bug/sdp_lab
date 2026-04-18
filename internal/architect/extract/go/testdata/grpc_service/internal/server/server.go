package server

import (
	"google.golang.org/grpc"
)

type GreeterServer struct {
	// UnimplementedGreeterServer would be embedded here
}

func (s *GreeterServer) SayHello(req interface{}) (interface{}, error) {
	return nil, nil
}

func RegisterGreeterServer(s *grpc.Server) {
	// Register the server
}
