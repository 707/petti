GOCACHE := $(CURDIR)/.gocache
GOMODCACHE := $(CURDIR)/.gomodcache
GOENV := GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)
VERSION ?= dev

.PHONY: test test-cover cover-check build build-release

test:
	$(GOENV) go test ./...
	npm test

test-cover:
	$(GOENV) go test -cover ./...

cover-check:
	$(GOENV) ./scripts/check-coverage.sh

build:
	$(GOENV) go build ./cmd/petti

build-release:
	$(GOENV) go build -ldflags "-X main.version=$(VERSION)" -o petti ./cmd/petti
