SHELL := /usr/bin/env bash

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PKG := github.com/jwmoss/skycli/internal/cli
GOFILES := $(shell find . -path './.git' -prune -o -path './.tools' -prune -o -name '*.go' -print)
LDFLAGS := -s -w -X $(PKG).version=$(VERSION) -X $(PKG).commit=$(COMMIT) -X $(PKG).date=$(DATE)

.PHONY: build fmt fmt-check test vet ci clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o skycli .

fmt:
	gofmt -w $(GOFILES)

fmt-check:
	test -z "$$(gofmt -l $(GOFILES))"

test:
	go test ./...

vet:
	go vet ./...

ci: fmt-check vet test build

clean:
	rm -rf skycli dist
