---
layout: post
author: Clement
title: Writing Protoc Plugins
categories: [Protocol Buffers, C++, Go]
---

Recently, I answered a [StackOverflow question](https://stackoverflow.com/questions/75343655/modeling-schema-metadata-without-serializing-into-the-protobuf-message/75362085#75362085) related to writing protoc plugins and Protobuf custom options. I thought this would be interesting to share how to write one because I believe this is quite an involved process and it fits the context of an article.

## C++ or Go

When checking the protobuf documentation, I could only find a plugin API for C++ and Go. Furthermore, Go seems to be the only language where people have written blog posts about how to write such a custom plugin. In this article, I'm trying to cover as many languages as possible so for now I'll write in both languages and if you find that another language support writing custom plugin, leave a comment and I'll be happy to update.

## Bazel

In order to build a multi-language project, I'm going to use Bazel. This might be frightening for some people but I'll try to explain as much as I can. Furthermore, if you are interested in learning Bazel, you can let me know in the comments.

## The Context

While the StackOverflow states the problem, I want to explain it again so that I have control of whether the content exists. Here is a copy of the question:

> Does protobuf support embedding non functional metadata into the protobuf schema without affecting the message serialization/de-serialization? I am attempting to embed business information (ownership, contact info) into a large shared protobuf schema but do NOT want to impact functionality at all.
>
> A structured comment or custom_option that does not get serialized would work. I would also like to parse the information from the .proto file for auditing purposes.
>
> TIA
>
> ```proto
> message Bar {
>  optional int32 a = 1 [(silent_options).owner = "team1", (silent_options).email = "team1@company.com"];
>  optional int32 b = 2;
>}
>```

In other words, we want to create a custom FieldOption which lets us assign an owner and an email to a field. On top of that, we want that to be analyzed for auditing purpose. This basically means that we can do that at "compile" time. So we are going to build a custom plugin which will let us write something like:

```shell
protoc --audit_out=. test.proto
```

Now, in this article, to keep everything simple, we are not going to generate any code or report stored in a file. We are going to print info on the terminal. However, generating files is pretty trivial to add in general. We simply write the information that we print in the terminal to a file (protoc library has some sort of printer to write to files).

### WORKSPACE + BUILD (root)

Let us first create our Bazel Workspace for our project. We do that by creating a WORKSPACE.bazel file at the root and inside we are going to add the dependencies needed to build our project.

```python WORKSPACE.bazel (Go) codeCopyEnabled
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

RULES_GO_VERSION = "0.37.0"
GO_VERSION = "1.19.5"
GAZELLE_VERSION = "0.29.0"
PROTOBUF_VERSION = "3.21.12"

# To create go libraries and binaries
http_archive(
  name = "io_bazel_rules_go",
  sha256 = "56d8c5a5c91e1af73eca71a6fab2ced959b67c86d12ba37feedb0a2dfea441a6",
  urls = [
    "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v%s/rules_go-v%s.zip" % (RULES_GO_VERSION, RULES_GO_VERSION),
    "https://github.com/bazelbuild/rules_go/releases/download/v%s/rules_go-v%s.zip" % (RULES_GO_VERSION, RULES_GO_VERSION),
  ],
)

# To generate BUILD.bazel files and lists of dependencies (more on that later)
http_archive(
  name = "bazel_gazelle",
  sha256 = "ecba0f04f96b4960a5b250c8e8eeec42281035970aa8852dda73098274d14a1d",
  urls = [
    "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v%s/bazel-gazelle-v%s.tar.gz" % (GAZELLE_VERSION, GAZELLE_VERSION),
    "https://github.com/bazelbuild/bazel-gazelle/releases/download/v%s/bazel-gazelle-v%s.tar.gz" % (GAZELLE_VERSION, GAZELLE_VERSION),
  ],
)

# To get protobuf and protoc libraries
http_archive(
  name = "com_google_protobuf",
  sha256 = "930c2c3b5ecc6c9c12615cf5ad93f1cd6e12d0aba862b572e076259970ac3a53",
  strip_prefix = "protobuf-%s" % PROTOBUF_VERSION,
  urls = [
    "https://mirror.bazel.build/github.com/protocolbuffers/protobuf/archive/v%s.tar.gz" % PROTOBUF_VERSION,
    "https://github.com/protocolbuffers/protobuf/archive/v%s.tar.gz" % PROTOBUF_VERSION,
  ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(version = GO_VERSION)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies(go_repository_default_config = "//:WORKSPACE.bazel")

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()
```
```python WORKSPACE.bazel (C++) codeCopyEnabled
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

PROTOBUF_VERSION = "3.21.12"

# To get protobuf and protoc libraries
http_archive(
  name = "com_google_protobuf",
  sha256 = "930c2c3b5ecc6c9c12615cf5ad93f1cd6e12d0aba862b572e076259970ac3a53",
  strip_prefix = "protobuf-%s" % PROTOBUF_VERSION,
  urls = [
    "https://mirror.bazel.build/github.com/protocolbuffers/protobuf/archive/v%s.tar.gz" % PROTOBUF_VERSION,
    "https://github.com/protocolbuffers/protobuf/archive/v%s.tar.gz" % PROTOBUF_VERSION,
  ],
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()
```

This gets all the dependencies needed to work with protobuf. Mainly we are going to use protobuf library which contains some generated code to deal with descriptors (a meta object that describe an object written in protobuf) and the protoc library which lets us define plugins.

#### Go

For go, we have some extra steps. The first thing we can do is creating our go module. To do that we can write the following command:

```shell codeCopyEnabled
go mod init test.com
```

Where you can replace `test.com` with the name of your module. **If you changed the module name, be aware that you'll need to update all the following `test.com`**.

Now, because we also want our application to run with a simple `go run main.go` kind of command, we are going to add a dependency to the module, which is protobuf. To do that enter the following command:

```shell codeCopyEnabled
go get -u google.golang.org/protobuf
```

Note that we added a protobuf dependency in the WORKSPACE.bazel and in our go.mod. These are not the same. One if for the building phase (linking with libraries) and the other is to be used in the Go program (as code).

Finally, we also need to set up Gazelle. We need to create a BUILD.bazel file at the root level.

```python BUILD.bazel codeCopyEnabled
load("@bazel_gazelle//:def.bzl", "gazelle")

# gazelle:prefix test.com
gazelle(name = "gazelle")

gazelle(
  name = "gazelle-update-repos",
  args = [
    "-from_file=go.mod",
    "-to_macro=deps.bzl%go_dependencies",
    "-prune",
  ],
  command = "update-repos",
)
```

This creates two commands (`gazelle` and `gazelle-update-repos`) that we can run to generate our BUILD.bazel and other dependency files automatically.

We can now run `bazel run //:gazelle-update-repos` in the terminal and we will see that it creates a file called `deps.bzl` and that the WORKSPACE.bazel was modified with these lines:

```python WORKSPACE.bazel
load("//:deps.bzl", "go_dependencies")

# gazelle:repository_macro deps.bzl%go_dependencies
go_dependencies()
```

If you open the `deps.bzl`, you will see a list of all the dependencies fetched to be able to build your go application.

### Protobuf

We are now at the stage where we can define our custom option. It is worth noting that in our case we need an option on fields but we can create options for a lot of different context. We could for example create an option at the top-level context (`go_package`, `optimize_for`, ...), at a message level, etc. You can find all the options in the file called descriptor.proto in the GitHub repo under `src/google/protobuf`.

To create a custom option, we need to extend the relevant message. In our case we need to extend `google.protobuf.FieldOptions`. To do that we can simply use the `extend` concept, which lets us define more fields inside an already existing message.

```proto proto/silent_option.proto (Go) codeCopyEnabled
syntax = "proto3";

import "google/protobuf/descriptor.proto";

option go_package = "test.com/proto";

message SilentOption {
  string owner = 1;
  string email = 2;
}

extend google.protobuf.FieldOptions {
  SilentOption silent_option = 1000; // see note below for why 1000
}
```

```proto proto/silent_option.proto (C++) codeCopyEnabled
syntax = "proto3";

import "google/protobuf/descriptor.proto";

message SilentOption {
  string owner = 1;
  string email = 2;
}

extend google.protobuf.FieldOptions {
  SilentOption silent_option = 1000; // see note below for why 1000
}
```

> NOTE: if you check the FieldOptions message in the descriptor.proto file, you will see the following line: `extensions 1000 to max;`. This means that when we are extending this message, our fields will need to contain tags that are between 1000 and max (maximum tag). Furthermore, some of the option tags are "already taken". This means that other custom options are using them and if you were to use your option with another one having the same tag, you would have a conflict. Check the list of the [Protobuf Global Extension Registry](https://github.com/protocolbuffers/protobuf/blob/main/docs/options.md) before selecting the tag for your custom option and maybe register it.

Now that we have our proto file, we can think about compiling it. To do that we are going to create a BUILD.bazel file in the proto directory. This will define a library for our proto file and another for the related programming language.

```shell BUILD.bazel (Go) codeCopyEnabled
bazel run //:gazelle
```

```python BUILD.bazel (C++) codeCopyEnabled
load("@rules_proto//proto:defs.bzl", "proto_library")
load("@rules_cc//cc:defs.bzl", "cc_proto_library")

proto_library(
  name = "silent_option_proto",
  srcs = ["silent_option.proto"],
  visibility = ["//visibility:public"],
  deps = ["@com_google_protobuf//:descriptor_proto"],
)

cc_proto_library(
  name = "silent_option_cc_proto",
  visibility = ["//visibility:public"],
  deps = [":silent_option_proto"],
)
```

#### Go

You might have noticed that we simply ran a command to generate our BUILD.bazel in the proto directory. This is what gazelle is doing. It checks your file and determine how to create BUILD files. However, I think there are problems with this solution. The main one is the naming of our libraries. By now, you should have something like this:

```python proto/BUILD.bazel
load("@rules_proto//proto:defs.bzl", "proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

proto_library(
  name = "proto_proto",
  srcs = ["silent_option.proto"],
  visibility = ["//visibility:public"],
  deps = ["@com_google_protobuf//:descriptor_proto"],
)

go_proto_library(
  name = "proto_go_proto",
  importpath = "test.com/proto",
  proto = ":proto_proto",
  visibility = ["//visibility:public"],
)

go_library(
  name = "proto",
  embed = [":proto_go_proto"],
  importpath = "test.com/proto",
  visibility = ["//visibility:public"],
)
```

and these are using generic names based on the folder there are stored in (proto). Let's rename all that.

```python proto/BUILD.bazel codeCopyEnabled
load("@rules_proto//proto:defs.bzl", "proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

proto_library(
  name = "silent_option_proto",
  #...
)

go_proto_library(
  name = "silent_option_go_proto",
  proto = ":silent_option_proto",
  #...
)

go_library(
  embed = [":silent_option_go_proto"],
  #...
)
```

### Plugin

Finally, we arrive at the moment where we need to write the plugin. The goal of this plugin is reading a protobuf file and if it sees a field with silent_option, it will print the file name, the field and its related info, and the option content.

> Note: Since this is a very different process for different implementations of Protobuf, I will rely on code comment to explain what the code is doing.

```go main.go codeCopyEnabled
package main

import (
  "flag"
  "log"

  "google.golang.org/protobuf/compiler/protogen"
  "google.golang.org/protobuf/proto"
  "google.golang.org/protobuf/types/descriptorpb"

  pb "test.com/proto"
)

func main() {
  var flags flag.FlagSet
  // defines the options that we can pass to our plugin
  team := flags.String("team", "", "Filtering team")

  protogen.Options{
    ParamFunc: flags.Set, // the protobuf library will set the option into the flags variable
  }.Run(func(gen *protogen.Plugin) error {
    for _, file := range gen.Files { // iterates over all the proto files given as source
      if !file.Generate {
        continue
      }

      for _, message := range file.Messages { // iterates over the messages in the current file
        for _, field := range message.Fields { // iterates over the fields in the current message
          option := field.Desc.Options().(*descriptorpb.FieldOptions) // try to get an option

          if option == nil { // if no option we skip
            continue
          }

          extension := proto.GetExtension(option, pb.E_SilentOption).(*pb.SilentOption) // try to cast this option in SilentOption

          if extension != nil && len(extension.Owner) != 0 && team != nil && extension.Owner == *team {
            // in here we have a SilentOption which as the owner equal to the team option pass in command line.
            log.Println(file.Desc.Name(), field, extension)
          }
        }
      }
    }
    return nil
  })
}
```

```cpp main.cc codeCopyEnabled
#include <string>
#include <google/protobuf/compiler/plugin.h>
#include <google/protobuf/compiler/code_generator.h>
#include <google/protobuf/descriptor.h>
#include <google/protobuf/io/printer.h>
#include <google/protobuf/compiler/command_line_interface.h>
#include "proto/silent_option.pb.h"

using namespace std;
using namespace google::protobuf;
using namespace google::protobuf::io;
using namespace google::protobuf::compiler;

// implementation of Generator interface
class AuditGenerator : public CodeGenerator {
 public:
  // iterates over the files and call the Generate function
  // we are skipping error handling
  virtual bool GenerateAll(
    const std::vector<const FileDescriptor*> &files,
    const std::string &parameter,
    GeneratorContext *generator_context,
    std::string *error
  ) const override {
    for (auto &&file : files) {
      this->Generate(file, parameter, generator_context, error);
    }
    return true;
  }

  // analyzes a file
  virtual bool Generate(
    const FileDescriptor *file,
    const std::string &parameter,
    GeneratorContext *generator_context,
    std::string *error
  ) const override {
    // iterates over the messages in the current file
    for (size_t i = 0; i < file->message_type_count(); ++i) {
      auto message = file->message_type(i);

      // iterates over the fields in the current message
      for (size_t j = 0; j < message->field_count(); ++j) {
        auto field = message->field(j);
        auto options = field->options();

        if (!options.HasExtension(silent_option)) {continue;} // if no SilentOption we skip

        auto extension = options.GetExtension(silent_option);

        if (extension.IsInitialized() && parameter.size() > 0 && extension.owner() == parameter) {
          // in here we have a SilentOption which as the owner equal to the team option pass in command line.
          std::cerr << file->name() << ": " << field->DebugString() << std::endl;
        }
      }
    }

    return true;
  }
};

int main(int argc, char *argv[]) {
  AuditGenerator generator;
  return PluginMain(argc, argv, &generator); // registers the generator
}
```

To compile this code, we need to create a BUILD.bazel file which will generate a binary for our application.

```shell BUILD.bazel (Go) codeCopyEnabled
bazel run //:gazelle
```

```python BUILD.bazel (C++) codeCopyEnabled
load("@rules_cc//cc:defs.bzl", "cc_binary")

cc_binary(
  name = "protoc-gen-audit",
  srcs = ["main.cc"],
  deps = [
    "//proto:silent_option_cc_proto",
    "@com_google_protobuf//:protobuf",
    "@com_google_protobuf//:protoc_lib",
  ],
)
```

#### Go

Same naming problem as the Protobuf section. Let us rename that.

```python BUILD.bazel
# gazelle related code ...

go_library(
  name = "protoc-gen-audit_lib",
  #...
)

go_binary(
  name = "protoc-gen-audit",
  embed = [":protoc-gen-audit_lib"],
  #...
)
```

### Running

We can now build our binaries by running:

```shell codeCopyEnabled
bazel build //:protoc-gen-audit
```

We also need to have a proto file to test our plugin.

```proto test.proto (Go) codeCopyEnabled
syntax = "proto3";

import "proto/silent_option.proto";

option go_package = "another_test.com";

message Bar {
  int32 a = 1 [
    (silent_option) = {
      owner: "team1",
      email: "team1@company.com"
    }
  ];
  int32 b = 2 [
    (silent_option) = {
      owner: "team2",
      email: "team2@company.com"
    }
  ];
}
```

```proto test.proto (C++) codeCopyEnabled
syntax = "proto3";

import "proto/silent_option.proto";

message Bar {
  int32 a = 1 [
    (silent_option) = {
      owner: "team1",
      email: "team1@company.com"
    }
  ];
  int32 b = 2 [
    (silent_option) = {
      owner: "team2",
      email: "team2@company.com"
    }
  ];
}
```

Now that we have our binaries in the `bazel-bin` directory, we can use them with protoc as plugins. To do so we use the `--plugin` option which takes the path of our binary and the option related to our plugin. For example, our plugin is called `protoc-gen-audit`, so now we can use the `--audit_out` option.

> Note: we also added a team flag in Go. This lets us use `--audit_opt=team=THE_TEAM_NAME`.

```shell Go codeCopyEnabled
protoc --plugin=protoc-gen-audit=$(PWD)/bazel-bin/protoc-gen-audit_/protoc-gen-audit --audit_out=. --audit_opt=team=team1 test.proto
```

```shell C++ codeCopyEnabled
protoc --plugin=protoc-gen-audit=$(PWD)/bazel-bin/protoc-gen-audit --audit_out=team1:. test.proto
```

You can now play with your plugin and test with other team names.

### Conclusion

Obviously, we can improve the solution in this post but the most important is that we saw that we can create custom options and protoc plugins. This can be interesting for compile time analysis or generating code. Finally, we saw that in this auditing use case, but we could use this in more advanced use cases (e.g.: [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway)).

**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**