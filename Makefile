GOCACHE := $(CURDIR)/.gocache
GOMODCACHE := $(CURDIR)/.gomodcache
GOENV := GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)
VERSION ?= dev

.PHONY: test test-cover cover-check build build-release

test:
	$(GOENV) go test ./...

test-cover:
	$(GOENV) go test -cover ./...

cover-check:
	$(GOENV) ./scripts/check-coverage.sh

build:
	$(GOENV) go build ./cmd/pkgview

build-release:
	$(GOENV) go build -ldflags "-X main.version=$(VERSION)" -o pkgview ./cmd/pkgview
