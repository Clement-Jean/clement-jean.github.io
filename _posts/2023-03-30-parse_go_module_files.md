---
layout: post
author: Clement
title: "Parse go module files"
categories: [Go]
---

Did you ever need to know, inside your program, on which go version you are running? That's what we are going to solve today. The most common use case for this is logging. We want to be able to debug by reproducing the environment of where the binary is running as close as possible. This starts by knowing which version of go we are using.

# Setup

To get started doing that, we will need a go module. Let's create that:

```shell
$ go mod init test.com
```

We should now have a go.mod file in our folder. If you inspect this file, we can get the go version which will be used for compiling the project. It looks like this:

```text go.mod
module test.com

go 1.20
```

That's pretty much it. We will use this file to get the info we want.

# Go Command

One thing that I learned recently is that we can actually get a JSON representation of or modules, workspaces, ... via the command line. To do that, we can run the following command:

```shell
$ go mod edit -json
{
  "Module": {
    "Path": "test.com"
  },
  "Go": "1.20",
  "Require": null,
  "Exclude": null,
  "Replace": null,
  "Retract": null
}
```

And we have our JSON!

# Parsing JSON

The only thing left to do is execute this command in our main, Unmarshal the JSON result and we should be able to get the version.

First, let's define the structs into which we will Unmarshal to.

```go main.go
type Module struct {
  Path string
}

type GoMod struct {
  Module Module
  Go     string
}
```

Notice here that I'm not taking Require, Exclude, ... into account. We only want the Go string.

After that, the rest is pretty easy. We can execute a command line and get its stdout result like so:

```go
out, _ := exec.Command("go", "mod", "edit", "-json").Output()
```

I'm skipping the err handling here by dropping the error with _ but make sure to handle this for production-grade scripts.

And finally, we use `json.Unmarshal` function which takes the ouput of the command line, and the destination of where we want to populate the data. In our case, this is an instance of GoMod.

In the end, the main function looks like:

```go main.go
package main

import (
  "encoding/json"
  "fmt"
  "os/exec"
)

type Module struct {
  Path string
}

type GoMod struct {
  Module Module
  Go     string
}

func main() {
  var mod GoMod
  out, _ := exec.Command("go", "mod", "edit", "-json").Output()

  if err := json.Unmarshal(out, &mod); err == nil {
    fmt.Println(mod.Go)
  }
}
```

We can run that, and we get:

```shell
$ go run main.go
1.20
```

We now have our Go version at runtime and we can use it for logging, selecting features, ...

# Conclusion

In this short post, we saw how to get the go version on which a go project is compiled at runtime. This can be used in multiple ways, but the use case that has come to me is mostly logging. I hope this is interesting for you.

**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**