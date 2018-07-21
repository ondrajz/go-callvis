VERSION := $(shell git describe --tags --always --dirty)
GOARCH  := $(shell go env GOARCH)
LDFLAGS := -X main.Version=$(VERSION)

BUILD_DIR ?= build
BINARY	  := go-callvis
RELEASE	  := $(BUILD_DIR)/$(BINARY)_$(VERSION)-$(GOARCH)

DEP ?= $(GOPATH)/bin/dep


all: $(DEP) install

$(DEP):
	go get -u github.com/golang/dep/cmd/dep
	dep version

install:
	@echo "-> Installing go-callvis $(VERSION)"
	go install -ldflags "$(LDFLAGS)"

build:
	@echo "-> Building go-callvis $(VERSION)"
	go build -v -ldflags "$(LDFLAGS)" -o $(BINARY)

release:
	@echo "-> Releasing go-callvis $(VERSION)"
	mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(RELEASE)

clean:
	go clean -i

test:
	go test -v


.PHONY: all install build release clean test
