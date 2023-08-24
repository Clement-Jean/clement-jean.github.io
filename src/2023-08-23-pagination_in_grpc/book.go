package main

import (
	"strings"
	"time"

	pb "github.com/Clement-Jean/clement-jean.github.io/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Book struct {
	ID          string `gorm:"primarykey"`
	Name        string `gorm:"size:255"`
	Description string `gorm:"size:255"`
	Authors     string `gorm:"size:255"`
	Published   time.Time
	Pages       int
	Isbn        string
}

func mapBookToBookPb(book Book) *pb.Book {
	return &pb.Book{
		Name:        book.Name,
		Description: book.Description,
		Authors:     strings.Split(book.Authors, ","),
		Published:   timestamppb.New(book.Published),
		Pages:       uint32(book.Pages),
		Isbn:        book.Isbn,
	}
}
