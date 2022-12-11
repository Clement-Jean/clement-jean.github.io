---
layout: post
author: Clement
title: gRPC Go Server Embeddings
categories: [gRPC, Go]
---

One of the common thing that my students are asking about recently is the difference between 2 Type Embeddings when your are defining a Server type for Service Registration. While this is an important topic, the gRPC doc seems to only mention that the `Unimplemented` version is for Forward Compatibility, and my course, up until now, uses the name of the generated Service Server directly. As such, I thought I would give an explanation on why I now recommend to use `Unimplemented` and some examples of the 3 Type Embeddings that you can use.

## Type Embedding

One thing that might not be clear for everyone is what is a Type Embedding and why we need it in gRPC. The first thing to understand is that Go is a language that uses composition instead of inheritance. And if you don't know about Composition or you just want a refresher, you friend Wikipedia is here: [Composition over Inheritance](https://en.wikipedia.org/wiki/Composition_over_inheritance).

On top of composition, Go allows anonymous fields in a struct. While I think anonymous field is a misnomer because the field can be referenced by the type name, these provide a shorter way (no need for Identifier) of writing composition. Let's take an example:

```go
type A struct {
	s string
}

type B struct {
	s string
	A // no identifier here, just a type
}

func main() {
	var b B

	b.s = "Test"
	b.A.s = "Another Test" // notice that we can access A even if it's 'anonymous'
	fmt.Println(b)
}
```
In this example, we augmented `B` with the fields defined in `A`. The ouput of this program should be something like: `{Test {Another Test}}` where the outter object is `B` and the inner object is `A`.

So in the end this is just a convenient way of writing composition.

## gRPC Go

Now, that we are clear on what is a Type Embedding, we can talk about its role in gRPC. As we know the protoc compiler will generate some code for our services, and we also know that services are contracts between a server and client. So basically, because we have a contract we need to make sure that this is implemented on both side of the wire.

So if we define a dummy service:

```proto
service DummyService { }
```

And we generate our code:

```shell
protoc --grpc-go_out=. dummy.proto 
```

We have the following generated server code (simplified):

```go
// DummyServiceServer is the server API for DummyService service.
// All implementations must embed UnimplementedDummyServiceServer
// for foward compatibility
type DummyServiceServer interface {
	mustEmbedUnimplementedDummyServiceServer()
}

// UnimplementedDummyServiceServer must be embedded to have forward compatible implementations.
type UnimplementedDummyServiceServer struct {
}

func (UnimplementedDummyServiceServer) mustEmbedUnimplementedDummyServiceServer() {}

// UnsafeDummyServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DummyServiceServer will
// result in compilation errors.
type UnsafeDummyServiceServer struct {
	mustEmbedUnimplementedDummyServiceServer()
}
```

First, we can notice a `mustEmbedUnimplementedDummyServiceServer` function. While I'm not entirely sure what this is doing since I can still compile without the `Unimplemented` embedding, I read on [Issue 3794](https://github.com/grpc/grpc-go/issues/3794) that `RegisterDummyService` will require (probrably in the future) the Server to embed the `UnimplementedDummyServiceServer`.

Then, as mentionned in the `DummyServiceServer` documentation, this is the server API. This means that when we add rpc endpoints to our service in the .proto file, methods will be generated into that interface.

The second type will always be empty. However, once we add rpc endpoints, a method will be added to this type and this method will simply return a gRPC error.

And finally, the last type will stay as is and no methods will be added to it.

## ${ServiceName}Server

This is the type embedding I used in my course. However, this is a mistake to use this directly. Let's see why.

Let's first add a rpc endpoint to our DummyService, this will help when we actually want to see the difference between the type embeddings by calling an endpoint.

```proto
import "google/protobuf/empty.proto";

service DummyService {
	rpc GetDummy(google.protobuf.Empty) returns (google.protobuf.Empty);
}
```

Then our Server type will look like this:

```go
type struct Server {
	DummyServiceServer
}
```

So, right now, we didn't implement `GetDummy` rpc endpoint. What happens if we try to call it ? The server runs perfectly, no compilation error, but once you call the rpc endpoint it will panic. This is where this type embedding is not Forward Compatible because an service which doesn't have a complete implementation of our service might cause a panic when comunicating with one that has the implementation.

## Unsafe${ServiceName}Server

Let's skip the `Unimplemented` for now and let's take a look at the `Unsafe` type emdeding. Before explaining it though, I want to mention two things:

- `Unsafe` sounds really bad. However in some specific cases, this embedding might actually be useful.
- The type documentation says that this type is not recommended, but once again, be aware that it might be useful.

With that said, let's get started. Let's replace our type embedding:

```go
type struct Server {
	UnsafeDummyServiceServer
}
```

In this case, calling an unimplemented endpoint will also result in a panic at runtime, but the main difference here is that this types embedding will help you to catch the unimplemented endpoints at compile time. This means that each time you add a rpc endpoint it will force you to implement it in your Go code. I actually like that approach more but the problem of panic at runtime is still here.

So in most of cases this is something you will not use because this is similar to the previous type embedding we showed. It will panic at runtime if a rpc endpoint is not defined. However, if you can control all your clients and servers, meaning that you can update all of them at the same time (and for eternity), this type embedding is actually safer (ironic, right ?) because it helps you to discover all the unimplemented rpc endpoint in your service at compile time.

## Unimplemented${ServiceName}Server

And now, here is the one that you should use in most of the cases. This type embedding, as mentionned earlier will get a default implementation for all rpc endpoint added in the service. This means that for the service that we defined earlier, we are going to have the following method generated:

```go
func (UnimplementedDummyServiceServer) GetDummy(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return status.Errorf(codes.Unimplemented, "method GetDummy not implemented")
}
```

And now we basically have Forward Compatibility because if a service without full implementation is called, it will just return a gRPC error and will not panic.

## Conclusion

In conclusion, you might have use cases where you actually need `Unsafe` type embedding but most of the time use the `Unimplemented` one. As for the other type embedding, forget it, there is no advantage in using it, only disadvantages. I hope this was helpful and see you in the next post.
