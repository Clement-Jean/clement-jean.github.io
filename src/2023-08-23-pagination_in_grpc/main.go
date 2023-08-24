package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	pb "github.com/Clement-Jean/clement-jean.github.io/proto"
)

func main() {
	hostDb := os.Getenv("POSTGRES_HOST")
	userDb := os.Getenv("POSTGRES_USER")
	pwdDb := os.Getenv("POSTGRES_PASSWORD")
	nameDb := os.Getenv("POSTGRES_DB")
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable", hostDb, userDb, pwdDb, nameDb)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("couldn't connect to the database: %v", err)
	}

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

	var opts []grpc.ServerOption
	s := grpc.NewServer(opts...)

	pb.RegisterBookStoreServiceServer(s, &server{
		db: db,
	})

	log.Printf("listening at %s\n", addr)

	defer s.Stop()
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}
