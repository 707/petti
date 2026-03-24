GOCACHE := $(CURDIR)/.gocache
GOMODCACHE := $(CURDIR)/.gomodcache
GOENV := GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)

.PHONY: test test-cover cover-check build

test:
	$(GOENV) go test ./...

test-cover:
	$(GOENV) go test -cover ./...

cover-check:
	$(GOENV) ./scripts/check-coverage.sh

build:
	$(GOENV) go build ./cmd/pkgview
