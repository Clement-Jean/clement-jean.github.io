package main

import (
	"log"
	"net"

	auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/grpc"

	pb "github.com/Clement-Jean/clement-jean.github.io/proto"
)

type Server struct {
	pb.UnimplementedGreetServiceServer
}

func main() {
	addr := "0.0.0.0:50051"
	lis, err := net.Listen("tcp", addr)

	if err != nil {
		log.Fatalf("failed to listen: %v\n", err)
	}

	defer func(lis net.Listener) {
		if err := lis.Close(); err != nil {
			log.Fatalf("unexpected error: %v", err)
		}
	}(lis)
	log.Printf("listening at %s\n", addr)

	var opts []grpc.ServerOption
	s := grpc.NewServer(opts...)

	srv := &Server{}
	auth.RegisterAuthorizationServer(s, srv)
	pb.RegisterGreetServiceServer(s, srv)

	defer s.Stop()
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}
