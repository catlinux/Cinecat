GO ?= go
BIN := bin/cinecatd
VERSION := $(shell cat VERSION)

.PHONY: build test run vet clean

build:
	$(GO) build -trimpath -ldflags "-X main.version=$(VERSION)" -o $(BIN) ./cmd/cinecatd

test:
	$(GO) test ./...

run:
	$(GO) run ./cmd/cinecatd

vet:
	$(GO) vet ./...

clean:
	rm -rf bin
