.PHONY: default install build test release

GOARCH := $(shell go env GOARCH)
VERSION := $(shell git describe --dirty)
LDFLAGS := -X main.Version=$(VERSION)

default: install

dep:
	dep ensure

get-deps:
	go get -u github.com/golang/dep/cmd/dep

clean:
	go clean -i

install: get-deps dep clean
	go install -ldflags "$(LDFLAGS)"

build:
	go build -v -ldflags "$(LDFLAGS)"

release:
	mkdir -p ./build/
	go build -v -ldflags "$(LDFLAGS)" -o build/go-callvis_$(VERSION)-$(GOARCH)

test:
	go test -v
