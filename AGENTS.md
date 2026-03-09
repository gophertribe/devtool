# Overview

`devtool` is a scaffolding tool for setting up go project builds using a custom binary `dev` build tool.

Each project using this pattern contains a `cmd/dev` application that compiles using a `Makefile` template found in this repository under `template/Makefile`. The `dev` application is a cobra-based CLI that tests/builds/packages/deploys software components.

`devtool` is itself a cobra CLI that has the following comands:

- `init` - sets up `cmd/dev` and `Makefile` for a given repository.

`github.com/gophertribe/devtool` contains helper packages that can be used to test and build applications in a standard way. It can use a Docker-based development environment to cross-compile binaries with cgo dependencies to different platforms.
