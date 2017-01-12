.PHONY: default install build test

VERSION := $(shell git describe --dirty)
LDFLAGS := -X main.Version=$(VERSION)

default: install

install:
	go clean -i
	go install -ldflags "$(LDFLAGS)"

build:
	go build -v -ldflags "$(LDFLAGS)"

test:
	go test -v ./...

get-deps:
	go get -t -v ./...
