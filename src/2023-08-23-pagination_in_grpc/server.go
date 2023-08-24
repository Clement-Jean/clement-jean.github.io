package main

import (
	"gorm.io/gorm"

	pb "github.com/Clement-Jean/clement-jean.github.io/proto"
)

type server struct {
	db *gorm.DB
	pb.UnimplementedBookStoreServiceServer
}
