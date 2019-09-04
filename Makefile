SHELL = /bin/bash

GOOS ?= $(shell go env GOOS)
GOARCH = amd64
BUILD_DIR ?= ./build
ORG := github.com/TrueFurby
PROJECT := go-callvis
REPOPATH ?= $(ORG)/$(PROJECT)

SUPPORTED_PLATFORMS := linux-$(GOARCH) darwin-$(GOARCH)
BUILD_PACKAGE = $(REPOPATH)

GIT_VERSION ?= $(shell git describe --always --tags --always --dirty)

GO_BUILD_TAGS := "mytag"
GO_LDFLAGS := "-X $(REPOPATH).commit=$(GIT_VERSION)"
GO_FILES := $(shell go list  -f '{{join .Deps "\n"}}' $(BUILD_PACKAGE) | grep $(ORG) | xargs go list -f '{{ range $$file := .GoFiles }} {{$$.Dir}}/{{$$file}}{{"\n"}}{{end}}')

$(BUILD_DIR)/$(PROJECT): $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH)
	cp $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH) $@

$(BUILD_DIR)/$(PROJECT)-%-$(GOARCH): $(GO_FILES) $(BUILD_DIR)
	GOOS=$* GOARCH=$(GOARCH) go build -tags $(GO_BUILD_TAGS) -ldflags $(GO_LDFLAGS) -o $@ $(BUILD_PACKAGE)

%.sha256: %
	shasum -a 256 $< &> $@

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PRECIOUS: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

.PHONY: cross
cross: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform).sha256)

.PHONY: release
release: cross
	@echo "releasing $(BUILD_DIR)/$(PROJECT)"
	ls -lA $(BUILD_DIR)

install:
	go install -tags $(GO_BUILD_TAGS) -ldflags $(LDFLAGS)

test: $(BUILD_DIR)/$(PROJECT)
	go test -v $(REPOPATH)

clean:
	rm -rf $(BUILD_DIR)

.PHONY: install test clean
