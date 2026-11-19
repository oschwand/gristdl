BINARY  := gristdl
MODULE  := github.com/oschwand/gristdl
VERSION := $(shell git describe --tags --always --dirty)
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build install test vet clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

install:
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" .

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
