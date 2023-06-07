package main

import (
	"fmt"

	pb "github.com/Clement-Jean/test/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func getOptionValue[T string | int | bool](
	opts *descriptorpb.MethodOptions,
	id protoreflect.ExtensionType,
) *T {
	value, ok := proto.GetExtension(opts, id).(T)

	if ok {
		return &value
	}

	return nil
}

func getServiceMethod(
	sd protoreflect.ServiceDescriptor,
	fn func(md protoreflect.MethodDescriptor) bool,
) *protoreflect.MethodDescriptor {
	for i := 0; i < sd.Methods().Len(); i++ {
		md := sd.Methods().Get(i)

		if fn(md) {
			return &md
		}
	}

	return nil
}

func getMethodOptionValue[T string | int | bool](
	sd protoreflect.ServiceDescriptor,
	id protoreflect.ExtensionType,
) *T {
	var value *T = nil

	getServiceMethod(sd, func(md protoreflect.MethodDescriptor) bool {
		opts, ok := md.Options().(*descriptorpb.MethodOptions)

		if !ok {
			return false
		}

		if tmp := getOptionValue[T](opts, id); tmp != nil {
			value = tmp
			return true
		}

		return false
	})

	return value
}

func getMethodExtension[T string | int | bool](
	fd protoreflect.FileDescriptor,
	id protoreflect.ExtensionType,
) *T {
	for i := 0; i < fd.Services().Len(); i++ {
		sd := fd.Services().Get(i)

		if value := getMethodOptionValue[T](sd, id); value != nil {
			return value
		}
	}

	return nil
}

func main() {
	world := getMethodExtension[string](pb.File_proto_world_proto, pb.E_Hello)

	if world != nil {
		fmt.Println(*world)
	}
}
