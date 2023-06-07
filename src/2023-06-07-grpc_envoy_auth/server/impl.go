package main

import (
	"context"
	"fmt"
	"net/http"

	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"

	auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	envoy_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	pb "github.com/Clement-Jean/clement-jean.github.io/proto"
)

func denied(code int32, body string) *auth.CheckResponse {
	return &auth.CheckResponse{
		Status: &status.Status{Code: code},
		HttpResponse: &auth.CheckResponse_DeniedResponse{
			DeniedResponse: &auth.DeniedHttpResponse{
				Status: &envoy_type.HttpStatus{
					Code: envoy_type.StatusCode(code),
				},
				Body: body,
			},
		},
	}
}

func allowed() *auth.CheckResponse {
	return &auth.CheckResponse{
		Status: &status.Status{Code: int32(codes.OK)},
		HttpResponse: &auth.CheckResponse_OkResponse{
			OkResponse: &auth.OkHttpResponse{
				HeadersToRemove: []string{"token"},
			},
		},
	}
}

func containsToken(key string) (bool, error) {
	if len(key) == 0 {
		return false, fmt.Errorf("empty key")
	}

	return (key == "authz"), nil
}

func (*Server) Check(ctx context.Context, req *auth.CheckRequest) (*auth.CheckResponse, error) {
	headers := req.Attributes.Request.Http.Headers
	ok, err := containsToken(headers["token"])

	if err != nil {
		return denied(
			http.StatusBadRequest,
			fmt.Sprintf("failed retrieving the api key: %v", err),
		), nil
	}

	if !ok {
		return denied(http.StatusUnauthorized, "unauthorized"), nil
	}

	return allowed(), nil
}

func (*Server) Greet(ctx context.Context, req *pb.GreetRequest) (*pb.GreetResponse, error) {
	return &pb.GreetResponse{}, nil
}
