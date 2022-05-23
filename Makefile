SHELL := /usr/bin/env bash -o pipefail

GIT_VERSION ?= $(shell git describe --always --tags --match 'v*' --dirty)
COMMIT     ?= $(shell git rev-parse HEAD)
BRANCH     ?= $(shell git rev-parse --abbrev-ref HEAD)
BUILD_DATE ?= $(shell date +%s)
BUILD_HOST ?= $(shell hostname)
BUILD_USER ?= $(shell id -un)

PROJECT := go-callvis
BUILD_DIR ?= .build

GOOS ?= $(shell go env GOOS)
GOARCH = amd64
PLATFORMS := linux-$(GOARCH) darwin-$(GOARCH)

GO_BUILD_TAGS ?= ""
GO_LDFLAGS := \
	-X main.commit=$(GIT_VERSION)
GO_FILES := $(shell go list ./... | xargs go list -f '{{ range $$file := .GoFiles }} {{$$.Dir}}/{{$$file}}{{"\n"}}{{end}}')

ifeq ($(NOSTRIP),)
GO_LDFLAGS += -w -s
endif

ifeq ($(NOTRIM),)
GO_BUILD_ARGS += -trimpath
endif

ifeq ($(V),1)
GO_BUILD_ARGS += -v
endif

export GO111MODULE=on
export DOCKER_BUILDKIT=1

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build:  ## Build go-callvis
	go build -tags $(GO_BUILD_TAGS) -ldflags "$(GO_LDFLAGS)" $(GO_BUILD_ARGS)

test:  ## Run unit tests
	go test -tags $(GO_BUILD_TAGS) -ldflags "$(GO_LDFLAGS)" $(GO_BUILD_ARGS) -short -race ./...

install:  ## Install go-callvis
	go install -tags $(GO_BUILD_TAGS) -ldflags "$(GO_LDFLAGS)" $(GO_BUILD_ARGS)

$(BUILD_DIR)/$(PROJECT): $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH)
	cp $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH) $@

$(BUILD_DIR)/$(PROJECT)-%-$(GOARCH): $(GO_FILES) $(BUILD_DIR)
	GOOS=$* GOARCH=$(GOARCH) go build -tags $(GO_BUILD_TAGS) -ldflags "$(GO_LDFLAGS)" -o $@ $(GO_BUILD_ARGS)

%.sha256: %
	shasum -a 256 $< &> $@

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PRECIOUS: $(foreach platform, $(PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

cross: $(foreach platform, $(PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform).sha256)

release: cross  ## Release go-callvis
	ls -hl $(BUILD_DIR)

clean:  ## Clean build directory
	rm -vrf $(BUILD_DIR)

.PHONY: help build test install cross release clean
