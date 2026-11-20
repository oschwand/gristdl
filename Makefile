BINARY  := gristdl

.PHONY: build install test vet clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BINARY) .

install:
	CGO_ENABLED=0 go install -trimpath -ldflags "-s -w" .

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
