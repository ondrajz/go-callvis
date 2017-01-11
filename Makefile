# go-callvis

VERSION := $(shell git describe --dirty)
LDFLAGS := -X main.Version=$(VERSION)

default: install

build:
	go build -v -ldflags "$(LDFLAGS)"

install:
	go install -ldflags "$(LDFLAGS)"
