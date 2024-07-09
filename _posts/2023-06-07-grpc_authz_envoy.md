---
layout: post
author: Clement
title: "Authorization with gRPC and Envoy"
categories: [Go, gRPC, Envoy]
---

Recently, I've been looking for a good alternative to [Traefik](https://traefik.io/traefik/) as Reverse Proxy for gRPC services. Traefik has great support for gRPC and other common features, but Envoy comes with Protobuf-backed configuration and even greater support for gRPC services. In the article, I want to show how you can make Envoy use your custom authorization logic before redirecting (or not) the request to other services.

## The code

The code is available [here](https://github.com/Clement-Jean/clement-jean.github.io/tree/working/src/2023-06-07-grpc_authz_envoy).

## Disclaimer

This post has been inspired from this [article](https://ekhabarov.com/post/envoy-as-an-api-gateway-authentication-and-authorization/). I thought it would be nice to have a little bit more details and explain how to run the whole thing.

## Envoy

If you don't know Envoy, it is a [Reverse Proxy](https://en.wikipedia.org/wiki/Reverse_proxy). This is basically a server relaying client requests to other servers (your services). It is generally used to protect the services from direct access and potential abuse. As such, Reverse Proxies can load balance, rate limit, and much more.

Envoy is a project originally designed by Lyft and it is described as "a high performance C++ distributed proxy designed for single services and applications, as well as a communication bus and “universal data plane” designed for large microservice “service mesh” architectures". That's a lot of buzz words! But for us, the most important is this feature:

> HTTP/2 AND GRPC SUPPORT
>
> Envoy has first class support for HTTP/2 and gRPC for both incoming and outgoing connections. It is a transparent HTTP/1.1 to HTTP/2 proxy.

One of the interesting features coming out of this support is the fact that Envoy can use custom gRPC services you develop. An example of this is the authorization service that I want to show you here.

## Protobuf

Before everything else, let us start by defining the service that we want to protect. Nothing fancy, we are going to use a simple GreetService:

```proto
syntax = "proto3";

package greet;

option go_package = "github.com/Clement-Jean/clement-jean.github.io/proto";

message GreetRequest {}
message GreetResponse {}

service GreetService {
  rpc Greet (GreetRequest) returns (GreetResponse);
}
```

After that, Envoy provides us with its own protobuf definition for authorization. We can take a look at [external_auth.proto](https://github.com/envoyproxy/envoy/blob/main/api/envoy/service/auth/v3/external_auth.proto) which contains the following:

```proto
// A generic interface for performing authorization check on incoming
// requests to a networked service.
service Authorization {
  // Performs authorization check based on the attributes associated with the
  // incoming request, and returns status `OK` or not `OK`.
  rpc Check(CheckRequest) returns (CheckResponse) {
  }
}
```

This means that we need to implement the Check unary endpoint and register the Authorization service.

## go-control-plane

Envoy provides us with a project called go-control-plane. This contains a collection of services such as Authorization that we can implement in our project.

To get it, we simply execute:

```shell
$ go get github.com/envoyproxy/go-control-plane
```

This gives us access to `github.com/envoyproxy/go-control-plane/envoy/service/auth/v3` which contains the following structs:

- CheckRequest
- CheckResponse
- AutorizationServer (which is an interface containing `Check`)

In our code we can now implement the `Check` function for our server type. This looks like this:

```go
import auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

func (*Server) Check(ctx context.Context, req *auth.CheckRequest) (*auth.CheckResponse, error) {
}
```

If you are familiar with gRPC, this should look familiar.

And to register the Authorization service we can simply register like we normally would:

```go
import (
  auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
  pb "github.com/Clement-Jean/clement-jean.github.io/proto"
)

type Server struct {
  pb.UnimplementedGreetServiceServer
}

func main() {
  //...
  srv := &Server{}
  auth.RegisterAuthorizationServer(s, srv)
  pb.RegisterGreetServiceServer(s, srv)
  //...
}
```

Notice that we are registering both services on the server here. This is not necessary. You could have a microservice specifically dedicated to authorization and the other one to greeting people.

## Envoy Configuration

Now, in order to make Envoy do what we want, we need to create some YAML configuration. This configuration generally contains the ports on which Envoy listens, filters for filtering requests based on some properties, and clusters which are a collection of one or more endpoints.

In our case we are going to create two clusters. One for the authorization service and the other service. This is in fact not necessary since we registered both services on the same server, but I wanted to show you that you can split clusters for different microservices.

The clusters definition looks like the following:

```yaml
clusters:
  - name: grpc_auth
    http2_protocol_options: {}
    lb_policy: round_robin
    load_assignment:
      cluster_name: grpc_auth
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 0.0.0.0
                port_value: 50051

  - name: grpc_greet
    http2_protocol_options: {}
    lb_policy: round_robin
    load_assignment:
      cluster_name: grpc_greet
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 0.0.0.0
                port_value: 50051
```

I hope we can agree on the fact that they are very similar so let's only dissect the authorization one.

`http2_protocol_options: {}` simply means that we are enabling HTTP/2 for this cluster. This is required for gRPC services since gRPC is basically Protobuf over HTTP/2.

Then we have `lb_policy: round_robin`. This is not required for us since we will have only one instance of each service but in the case you scale things up, you will have to balance the load across the multiple services.

And finally, all the rest is basically defining a cluster with the name `grpc_auth` which can be reached at the address `0.0.0.0:50051`.

Now that we have that, we can take a look at the listener. Let's see the first part of the listener which is simply defining on which address and port Envoy will listen.

```yaml
listeners:
- name: listener_grpc
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 50050
```

Once again Envoy will listen on `0.0.0.0:50050`. Now, note that even if you had `0.0.0.0:50051` there will be no conflict with the `0.0.0.0:50051` address set in the cluster. This is because generally the gRPC server will be containerized separately from Envoy and thus will listen on its own 50051 port.

Finally, things get a little bit more interesting when we talk about the filters. We need to start with a `http_connection_manager` that defines the route that we want to protect with authorization.

```yaml
listeners:
  - name: listener_grpc
    # address
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: grpc_json
          codec_type: AUTO
          route_config:
            name: route
            virtual_hosts:
            - name: vh
              domains: ["*"]
              routes:
              - match: { prefix: "/greet.GreetService/", grpc: {} }
                route: { cluster: grpc_greet, timeout: { seconds: 60 } }
```

The most important part is the virtual_hosts one. We say that we will accept requests from any domain (not recommended in prod), and then we basically that every request made on route matching `/greet.GreetService/` will be redirected to the `grpc_greet` cluster

After that, we will configure the external authorization.

```yaml
listeners:
  - name: listener_grpc
    # address
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          # route matching
          http_filters:
          - name: envoy.filters.http.ext_authz
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz
              grpc_service:
                envoy_grpc:
                  cluster_name: grpc_auth
                timeout: 0.5s
              transport_api_version: V3
              failure_mode_allow: false
              with_request_body:
                max_request_bytes: 8192
                allow_partial_message: true
                pack_as_bytes: true
              status_on_error:
                code: 503
```

The most important things in this part of the config are:

- `cluster_name: grpc_auth`. We are specifying that the Authorization service can be found in the `grpc_auth` cluster.
- `code: 503`. If any error happens such as not finding the `Check` endpoint, Envoy will return a 503 error code.
- `with_request_body` forwards HTTP body to the Authorization service.

Finally, we need to tell Envoy to actually route the requests. We simply do that by adding a `envoy.filters.http.router` at the end of the `http_filters`.

```yaml
listeners:
  - name: listener_grpc
    # address
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          # route matching
          http_filters:
          # ext_authz
          - name: envoy.filters.http.router
            typed_config:
              '@type': type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
```

## Let's `Check`

Now that we have our Envoy config ready, it is time to implement the `Check` endpoint. We will first create a small demo environment where we will receive a token as header. We will check:

- If the token is empty/doesn't exist -> Deny
- If the token value is different from 'authz' -> Deny
- Otherwise -> Allow

Token checks would normally involve a database of some sort but here, as this is a small demo, let's create a simple function.

```go
func containsToken(key string) (bool, error) {
  if len(key) == 0 {
    return false, fmt.Errorf("empty key")
  }

  return (key == "authz"), nil
}
```

Nothing fancy.

Now, in `Check` we will be receiving headers, not as metadata, but as part of the request. I will let you check [external_auth.proto](https://github.com/envoyproxy/envoy/blob/main/api/envoy/service/auth/v3/external_auth.proto) and [attribute_context.proto](https://github.com/envoyproxy/envoy/blob/main/api/envoy/service/auth/v3/attribute_context.proto) to understand a little bit more about the data that we will receive as part of `CheckRequest`.

So we now get the token from the headers and pass it through `containsToken`.

```go
func (*Server) Check(ctx context.Context, req *auth.CheckRequest) (*auth.CheckResponse, error) {
  headers := req.Attributes.Request.Http.Headers
  ok, err := containsToken(headers["token"])
  //...
}
```

Finally, we still have to do error handling. `ok` will tell us whether the key is in the 'database' and the `err` will be errors like `empty key`. Now, returning an error with `go-control-plane` and Envoy is a little bit different that what you might expect. This is because instead of returning the error as status like we normally do in gRPC is not compatible with Envoy. Instead, we need to return the status as part of the `CheckResponse`.

Two helpers functions aiming at creating a `Allow` and `Deny` response, that I found [here](https://ekhabarov.com/post/envoy-as-an-api-gateway-authentication-and-authorization/), are pretty self describing:

```go
import (
  //...
  "google.golang.org/genproto/googleapis/rpc/status"
  "google.golang.org/grpc/codes"

  envoy_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
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
```

Notice that we are not using the traditional `google.golang.org/grpc/status` here. We are using `google.golang.org/genproto/googleapis/rpc/status`. **As of the time of writing this, I'm not aware of why this is the case. I might come back and update that when I learned why.**

Finally, we can finish the implementation of both `Check` and `Greet`. We will make `Greet` return an empty response. And we will make `Check` return the result of `denied` in case of error and wrong token, or return the result of `allowed` if everything goes well.

```go
import (
  "net/http"
  //...
)

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
```

## Demo Time!

Here we are! It's demo time baby!

To test all of this, we will run our server:

```shell
$ go run server/*.go
listening at 0.0.0.0:50051
```

After that, let's use [func-e](https://func-e.io/) to run our Envoy instance:

```shell
$ func-e run -c envoy/config.yaml
```

And finally, I will use [grpcurl](https://github.com/fullstorydev/grpcurl) to query the Greet endpoint on `0.0.0.0:50050` (Envoy listener). Let's start with the happy path scenario:

```shell
$ grpcurl -plaintext \
          -proto proto/greet.proto \
          -rpc-header="token: authz" \
          0.0.0.0:50050 greet.GreetService/Greet
{

}
```

We get an empty `GreetResponse`, as expected.

Now, we can try without token:

```shell
$ grpcurl -plaintext \
          -proto proto/greet.proto \
          0.0.0.0:50050 greet.GreetService/Greet
ERROR:
  Code: Internal
  Message: failed retrieving the api key: empty key
```

We get an Internal error with the message "empty key".

And finally, we test with a wrong token:

```shell
$ grpcurl -plaintext \
          -proto proto/greet.proto \
          -rpc-header="token: authd" \
          0.0.0.0:50050 greet.GreetService/Greet
ERROR:
  Code: Unauthenticated
  Message: unauthorized
```

Unauthorized! Great everything is working as expected.

## Conclusion

In this post we saw that we can use Envoy to sit between our services and call an Authorization service to decide whether or not to forward the request to a given route. In our case, we worked on a simple token checking logic but this should look similar for your real-life scenario. I hope this was interesting, thank you for reading!

**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**
