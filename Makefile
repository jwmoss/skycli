SHELL := /usr/bin/env bash

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GORELEASER_VERSION ?= v2.15.4
PKG := github.com/jwmoss/skycli/internal/cli
GOFILES := $(shell find . -path './.git' -prune -o -path './.tools' -prune -o -name '*.go' -print)
LDFLAGS := -s -w -X $(PKG).version=$(VERSION) -X $(PKG).commit=$(COMMIT) -X $(PKG).date=$(DATE)
GORELEASER := go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

.PHONY: build fmt fmt-check test vet ci live-readonly-smoke release-check release-snapshot clean

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

live-readonly-smoke: build
	SKYCLI_BIN=./skycli scripts/live-readonly-smoke.sh

release-check:
	$(GORELEASER) check

release-snapshot:
	$(GORELEASER) release --snapshot --clean

clean:
	rm -rf skycli dist
