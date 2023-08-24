package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Clement-Jean/clement-jean.github.io/proto"
	"github.com/Clement-Jean/clement-jean.github.io/utils"
)

const (
	defaultPageSize = 10
	maxPageSize     = 30
)

func validatePageSize(req *pb.ListBooksRequest) error {
	if req.PageSize > maxPageSize {
		msg := fmt.Sprintf(
			"expected page size between 0 and %d, got %d",
			maxPageSize, req.PageSize,
		)
		return errors.New(msg)
	} else if req.PageSize == 0 {
		req.PageSize = defaultPageSize
	}

	return nil
}

func (s *server) ListBooks(ctx context.Context, req *pb.ListBooksRequest) (*pb.ListBooksResponse, error) {
	if err := validatePageSize(req); err != nil {
		return nil, status.New(codes.InvalidArgument, err.Error()).Err()
	}

	if _, err := ulid.Parse(req.PageToken); len(req.PageToken) != 0 && err != nil {
		msg := fmt.Sprintf("expected valid ULID, got error %v", err)
		return nil, status.New(codes.InvalidArgument, msg).Err()
	}

	// build sql statement
	query := s.db.Table("book").Limit(int(req.PageSize)).Order("id ASC")

	if len(req.PageToken) != 0 {
		query = query.Where("id > ?", req.PageToken)
	}

	var queryRes = []Book{}
	query.Scan(&queryRes) // execute query

	if len(queryRes) == 0 {
		// short circuit if not results
		return &pb.ListBooksResponse{}, nil
	}

	// map to DTO and create response
	books := utils.Map(queryRes, mapBookToBookPb)
	lastItemIdx := len(queryRes) - 1
	nextPageToken := queryRes[lastItemIdx].ID

	if len(queryRes) < int(req.PageSize) {
		// no more pages
		nextPageToken = ""
	}

	return &pb.ListBooksResponse{
		Books:         books,
		NextPageToken: nextPageToken,
	}, nil
}
