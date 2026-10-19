.PHONY: clean install
VERSION ?= $(shell git describe --tags --always --dirty)
GIT_COMMIT ?= $(shell git rev-parse HEAD)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS = -X github.com/wandering-eye/nib.VERSION=$(VERSION) \
	-X github.com/wandering-eye/nib.GIT_COMMIT=$(GIT_COMMIT) \
	-X github.com/wandering-eye/nib.BUILD_DATE=$(BUILD_DATE)

dist/nib: clean
	go build -ldflags "$(LDFLAGS)" -o dist/nib cmd/nib/main.go
	go build -ldflags "$(LDFLAGS)" -o dist/dstask-import cmd/dstask-import/main.go

clean:
	rm -rf dist

install:
	cp dist/nib /usr/local/bin
	cp dist/dstask-import /usr/local/bin

test:
	go test -v ./...
	./integrationtest.sh

lint:
	golangci-lint run

update_deps:
	go get
