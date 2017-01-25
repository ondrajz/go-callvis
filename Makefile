.PHONY: default install build test release

GOARCH := $(shell go env GOARCH)
VERSION := $(shell git describe --dirty)
LDFLAGS := -X main.Version=$(VERSION)

default: install

install:
	go clean -i
	go install -ldflags "$(LDFLAGS)"

build:
	go build -v -ldflags "$(LDFLAGS)"

release:
	mkdir -p ./build/
	go build -v -ldflags "$(LDFLAGS)" -o build/go-callvis_$(VERSION)-$(GOARCH)

test:
	go test -v ./...

get-deps:
	go get -t -v ./...
