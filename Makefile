SHELL = /bin/bash

GIT_VERSION ?= $(shell git describe --always --tags --always --dirty)

GOOS ?= $(shell go env GOOS)
GOARCH = amd64
PLATFORMS := linux-$(GOARCH) darwin-$(GOARCH)

BUILD_DIR ?= ./build
ORG := github.com/ofabry
PROJECT := go-callvis
REPOPATH ?= $(ORG)/$(PROJECT)
BUILD_PACKAGE = $(REPOPATH)

GO_BUILD_TAGS ?= ""
GO_LDFLAGS := "-X main.commit=$(GIT_VERSION)"
GO_FILES := $(shell go list  -f '{{join .Deps "\n"}}' $(BUILD_PACKAGE) | grep $(ORG) | xargs go list -f '{{ range $$file := .GoFiles }} {{$$.Dir}}/{{$$file}}{{"\n"}}{{end}}')

export GO111MODULE=on

install:
	go install -tags $(GO_BUILD_TAGS) -ldflags $(GO_LDFLAGS)

$(BUILD_DIR)/$(PROJECT): $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH)
	cp $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH) $@

$(BUILD_DIR)/$(PROJECT)-%-$(GOARCH): $(GO_FILES) $(BUILD_DIR)
	GOOS=$* GOARCH=$(GOARCH) go build -tags $(GO_BUILD_TAGS) -ldflags $(GO_LDFLAGS) -o $@ $(BUILD_PACKAGE)

%.sha256: %
	shasum -a 256 $< &> $@

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PRECIOUS: $(foreach platform, $(PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

cross: $(foreach platform, $(PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform).sha256)

release: cross
	ls -hl $(BUILD_DIR)

test: $(BUILD_DIR)/$(PROJECT)
	go test -v $(REPOPATH)

clean:
	rm -rf $(BUILD_DIR)

.PHONY: cross release install test clean
