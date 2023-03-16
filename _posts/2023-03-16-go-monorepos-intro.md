---
layout: post
author: Clement
title: "Go Monorepos - Intro"
categories: [Go, Bazel]
---

Recently I've discovered two interesting was in creating monorepos for go projects. In this article we are going to talk about the advantages and disadvantages of each of these techniques.

This article is an introduction. I will skip over details here because the setup is generally dependent on the technologies you use. **If this article makes you want to know more about the subject, I invite you to tell me with which technology I should set a monorepo for (e.g. gRPC)**.

# Go Workspaces

The first of these techniques are using Go workspaces. This is nice to be able to do that without any other technology than Go. The goal here is basically to create submodules and link them to the workspace. Let's see an example.

Let's say that we have a server and a client. We will then have the following file architecture:

```
.
├── client
└── server
```

with that we can start initializing our modules. You can either go into each folder and run `go mod init <module_name>`, or you can automatize the process and run something like:

```shell Linux/MacOS
find . -maxdepth 1 -type d -not -path . -execdir sh -c "pushd {}; go mod init '<module_name>/{}'; popd" ";"
```
```shell Windows (Powershell)
Get-ChildItem . -Name -Directory | ForEach-Object { Push-Location $_; go mod init "<module_name>/$_" ; Pop-Location }
```

These commands will enter each directory, run `go mod init`, and get out of the directories. **Be careful though, if you have other folders that you don't want to use as modules, you will have to create more complex commands**.

Once we have this, we can create our workspace. To do that, we simply run:

```shell
go work init client server
```

And that's basically it. Client and Server and individual projects that can have their own set of dependencies and you can run each of them at the root folder by running:

```shell
go run ./server
```

and

```shell
go run ./client
```

## Advantages

- This is pretty low-tech. We only need Go.
- Pretty quick setup for new projects.
- Each of the module can get their own dependencies and they can also share some.

## Disadvantages

- All the subprojects need to be written in Go for it to work as intended.
- Setup for already existing and complex projects might be hard.

# Bazel

Because setting up a project in Bazel is highly dependent on which technology you want to use, I will not go into to much details here. But the idea with Bazel is that we can have the same kind of monorepo as we saw but we can do this is multi-languages.

If we have a client written in JS and a backend written in Go, we will have a BUILD.bazel file for each subprojects defining how to build each part of the projects. And at the root level we will have a WORKSPACE.bazel file which describes all the build dependencies.

Finally, if you are working with Go modules, [Gazelle](https://github.com/bazelbuild/bazel-gazelle) (a BUILD.bazel file generator) can help you write all the build boilerplate for you and let you focus on your code.

## Advantages

- Once it is set up, it is very efficient to run commands.
- Multi-language monorepos.
- Setup for already existing and complex projects is easier with Gazelle. Note that this is only for Go.

## Disadvantages

- Harder upfront cost for setup.

# Conclusion

In this article, we got an overview of two ways of building monorepos in Go or in multi-language setups. We saw that when we have a newer project we can use Go Workspaces as this is a low-tech way of starting building a monorepo. However, if we decide to have a multi-language setup we will have to move to Bazel. **There are obviously more choices to be made depending on the kind of project you are making and that's why I want your feedback on what I should build**.

**If you like this kind of content let me know in the comment section and feel free to ask for help on similar projects, recommend the next post subject or simply send me your feedback.**