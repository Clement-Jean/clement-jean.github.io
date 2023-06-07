---
layout: post
author: Clement
title: "Custom RPC Options in Protobuf"
categories: [Protobuf]
---

Recently I had to design authentication for a Blazor Application. After finishing implementing, I soon faced the need to know which RPC endpoint needs authentication and which doesn't. And while part of the problem is a solved one, I still needed a mechanism to let me define this. Let's see how.

> All the code (**only running through Bazel right now**) is available <a href="https://github.com/Clement-Jean/clement-jean.github.io/tree/working/src/2023-04-17-custom_rpc_options">here</a>

## Custom Options

Before explaining what my solution to the problem is, I'd like to make sure you understand what are custom options in Protobuf and how to define one. If you are confident about this skill, feel free to skip to the next section.

**A custom option is a way to define metadata for a proto file, message, enum, fields, service and rpc**. Generally, we are used to these:

```proto
option go_package = "github.com/Clement-Jean/test";
```

being placed at the top of the proto file. But it is important to know that you can make a field or message deprecated like so:

```proto
message Test {
  option deprecated = true;

  int32 field = 1 [ deprecated = true ];
}
```

Now, I agree that, in most of the cases, these option are more informational than anything else. They do not necessarily impact the code generation but they are here to document the code. Also, knowing that Protobuf has reflection, we can use them in our code. This means that we could have a tool checking for deprecated messages, fields, ... and give us warnings when we use them in our code base.

How do we define one, though? Well, it turns out that this is pretty simple. We need to use the `extend` concept and define which kind of option we want to extend. Let's first take a look at what kind of options we have:

```proto descriptor.proto
message FileOptions {
  //...

  // Clients can define custom options in extensions of this message. See above.
  extensions 1000 to max;
}

message MessageOptions {
  //...
  extensions 1000 to max;
}

message FieldOptions {
  //...
  extensions 1000 to max;
}

message OneofOptions {
  //...
  extensions 1000 to max;
}

message EnumOptions {
  //...
  extensions 1000 to max;
}

message EnumValueOptions {
  //...
  extensions 1000 to max;
}

message ServiceOptions {
  //...
  extensions 1000 to max;
}

message MethodOptions {
  //...
  extensions 1000 to max;
}
```

That's actually every concept that we have in Protobuf! So let's define a simple option now. We will define an option called `hello` of type string. And for making this related to the problem that I'm trying to solve, let's define that option in `MethodOptions` which represents the options for RPC endpoints.

So we will extend `MethodOptions`:

```proto hello.proto
syntax = "proto3";

import "google/protobuf/descriptor.proto";

extend google.protobuf.MethodOptions { }
```

And then inside this `extend` we can just write the hello field:

```proto hello.proto
extend google.protobuf.MethodOptions {
  string hello = ??;
}
```

But what is the tag that we need to use? Well, if you noticed in the `descriptor.proto` we have an extension range. These are the numbers we can use for tag. For now, we will use 1000, however, be aware that some of these tags are reserved by some already defined options. **So if you were to use a tool that defines options that have the same tag number, there would be conflicts**.

We now have:

```proto hello.proto
extend google.protobuf.MethodOptions {
  string hello = 1000;
}
```

## Reflection

Let us now use that option and read the value in code.

To use it, this is pretty simple, we just need to import the file in which we wrote the `extend` and make sure we use the option on an RPC endpoint.

```proto world.proto
syntax = "proto3";

import "hello.proto";

//...

service HelloWorldService {
  rpc HelloWorld (HelloWorldRequest) returns (HelloWorldResponse) {
    option (hello) = "world";
  };
}
```

We can generate the proto files out of world.proto and hello.proto. And after that we can take a bottom-up approach to read this value through reflection. By bottom up, I mean that we are going to first see how to read the value of a `MethodOptions`, then we will go to getting a `MethodDescriptor` out of a `ServiceDescriptor`, and finally getting a `ServiceDescriptor` out a `FileDescriptor`.

### Getting an Option Value

The first thing we are going to deal with is `MethodOptions`. These represent the options set on an RPC endpoint. In most of the implementations, we can check the existence of an option so this is as simple as "check if there is the option with a given id on this method, if yes return the value, otherwise return null".

```go main.go
import (
  "google.golang.org/protobuf/proto"
  "google.golang.org/protobuf/reflect/protoreflect"
  "google.golang.org/protobuf/types/descriptorpb"
)

func getOptionValue[T string | int | bool]( // T is not covering all types...
  opts *descriptorpb.MethodOptions,
  id protoreflect.ExtensionType,
) *T {
  value, ok := proto.GetExtension(opts, id).(T)

  if ok {
    return &value
  }

  return nil
}
```
```cpp main.cc
#include <optional>

using namespace google::protobuf;

template<typename OPT_T>
std::optional<OPT_T> get_option_value(
  const MethodOptions &opts,
  const auto &id
) {
  return opts.HasExtension(id) ?
    std::optional(opts.GetExtension(id)) :
    std::nullopt;
}
```
```java main.java
import com.google.protobuf.DescriptorProtos;
import com.google.protobuf.GeneratedMessage;
import java.util.Optional;

private static <T> Optional<T> getOptionValue(
  DescriptorProtos.MethodOptions opts,
  GeneratedMessage.GeneratedExtension<DescriptorProtos.MethodOptions, ?> id
) {
  return opts.hasExtension(id) ?
    Optional.of((T)opts.getExtension(id)) :
    Optional.empty();
}
```
```python main.py
def get_option_value(opts, id):
  for field in opts.ListFields():
    (desc, value) = field

    if value != "" and desc.name == id:
      return value
```
```csharp main.cs
using pb = global::Google.Protobuf;

static private T GetOptionValue<T>(
  this pb.Reflection.MethodDescriptor md, // MethodDescriptor and not MethodOptions as promised (sorry!)
  pb::Extension<pb.Reflection.MethodOptions, T> id
) => md.GetOptions().GetExtension(id);
```

### Getting a Method

The next step is to get a `MethodDescriptor` out of a `ServiceDescriptor`. This is done so that we can later call the GetOptionValue function on the options of that method (if any). We will basically loop over all the methods of a service and check for a predicate on each. If the predicate returns true, we "select" that method.

```go main.go
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
```
```cpp main.cc
#include <functional>

std::optional<const MethodDescriptor *> get_service_method(
  const ServiceDescriptor *sd,
  const std::function<bool(const MethodDescriptor *)> &predicate
) {
  if (!sd)
    return std::nullopt;

  for (int i = 0; i < sd->method_count(); ++i) {
    auto md = sd->method(i);

    if (predicate(md))
      return md;
  }

  return std::nullopt;
}
```
```java main.java
import com.google.protobuf.Descriptors;

private static <T> Optional<Descriptors.MethodDescriptor> getServiceMethod(
  Descriptors.ServiceDescriptor sd,
  Function<Descriptors.MethodDescriptor, Boolean> predicate
) {
  for (int i = 0; i < sd.getMethods().size(); ++i) {
    Descriptors.MethodDescriptor method = sd.getMethods().get(i);

    if (predicate.apply(method))
      return Optional.of(method);
  }

  return Optional.empty();
}
```
```python main.py
def get_service_method(sd, predicate):
  for method_name in sd.methods_by_name:
    md = sd.methods_by_name[method_name]

    if predicate(md):
      return md
```
```csharp main.cs
static private IEnumerable<pb.Reflection.MethodDescriptor> GetServiceMethod(
  this IEnumerable<pb.Reflection.ServiceDescriptor> services,
  Func<pb.Reflection.MethodDescriptor, bool> predicate
) => from svc in services
     from method in svc.Methods
     where predicate(method)
     select method;
```

### Putting Everything Together

And finally the idea is to call `GetServiceMethod` on all the `ServiceDescriptor`s and with the predicate is true we can call `GetOptionValue` on the method selected.

```go main.go
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
```
```cpp main.cc
template<typename U>
std::optional<U> get_method_option_value(
  const ServiceDescriptor *sd, // in C++ we can access the ServiceDescriptor directly
  const auto &id
) {
  if (!sd)
    return std::nullopt;

  std::optional<U> value;

  get_service_method(sd, [&value, &id](const MethodDescriptor *md) -> bool {
    auto opts = md->options();

    if (auto tmp = get_option_value<U>(opts, id))
      value = tmp;

    return value != std::nullopt;
  });

  return value;
}
```
```java main.java
private static <T> Optional<T> getMethodExtension(
  Descriptors.FileDescriptor fd,
  GeneratedMessage.GeneratedExtension<DescriptorProtos.MethodOptions, ?> id
) {
  for (int i = 0; i < fd.getServices().size(); ++i) {
    Descriptors.ServiceDescriptor sd = fd.getServices().get(i);
    Optional<T> world = getMethodOptionValue(sd, Hello.hello);

    if (world.isPresent())
      return world;
  }

  return Optional.empty();
}
```
```python main.py
def get_method_option_value(sd, id):
  md = get_service_method(sd, lambda md: get_option_value(md.GetOptions(), id) != None)

  return get_option_value(md.GetOptions(), id)

def get_method_extension(fd, id):
  for svc_name in fd.services_by_name:
    sd = fd.services_by_name[svc_name]
    value = get_method_option_value(sd, id)

    if value != None:
      return value
```
```csharp main.cs
static private T GetMethodOptionValue<T>(
  this pb.Reflection.FileDescriptor fd,
  pb::Extension<pb.Reflection.MethodOptions, T> id
) => fd.Services
       .GetServiceMethod(md => md.GetOptionValue(id) != null)
       .FirstOrDefault()
       .GetOptionValue(id);
```

### Usage

Let's see how to use that in our main function.

```go main.go
import "fmt"

func main() {
  // pb.File_proto_world_proto is the generated FileDescriptor
  // and pb.E_Hello the generated custom option
  world := getMethodExtension[string](pb.File_proto_world_proto, pb.E_Hello)

  if world != nil {
    fmt.Println(*world)
  }
}
```
```cpp main.cc
#include <iostream>

int main() {
  // HelloWorldService is the generated service
  auto sd = HelloWorldService::descriptor();
  // hello is the generated custom option
  auto world = get_method_option_value<std::string>(sd, hello);

  if (world)
    std::cout << *world << std::endl;
  return 0;
}
```
```java main.java
public static void main(String[] args) {
  // World is the FileDescriptor for the file world.proto
  Descriptors.FileDescriptor fd = World.getDescriptor();
  // Hello.hello is the generated custom option
  Optional<String> world = getMethodExtension(fd, Hello.hello);

  world.ifPresent(w -> System.out.println(w));
}
```
```python main.py
from proto.world_pb2 import DESCRIPTOR # FileDescriptor for world.proto

print(get_method_extension(DESCRIPTOR, "hello"))
```
```csharp main.cs
static public void Main(String[] args)
{
  // HelloExtensions.Hello is the generated custom option
  var id = HelloExtensions.Hello;
  // WorldReflection.Descriptor is the FileDescriptor for the file world.proto
  string world = WorldReflection.Descriptor.GetMethodOptionValue(id);

  if (world.Length != 0)
    System.Console.WriteLine(world);
}
```


## Back to the Problem

Now, as I mentioned I was trying to detect which routes need authentication with the help of such a custom option. It is not that hard to imagine the code we saw in the previous section work for an extension like the following:

```proto
extend google.protobuf.MethodOptions {
  bool is_authenticated = 1000;
}
```

Then, we can use it like so:

```proto
service CheckoutService {
  rpc Checkout (CheckoutRequest) returns (CheckoutResponse) {
    option (is_authenticated) = true;
  };
}
```

And with that we can get the value from the code we wrote earlier. We just need to be requesting the methods with Option having the id "is_authenticated" and make sure that we are asking for a boolean instead of a string.

## Conclusion

While this is a little bit hard to work directly with the Protobuf library, automating simple tasks like checking which routes need authentication save a lot of effort. I hope that you will find this content interesting and that you will share some of the extensions that you wrote.

**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**