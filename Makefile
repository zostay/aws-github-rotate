.PHONY: all
all: generate fmt test analyze

.PHONY: analyze
anaylyze:
	golangci-lint run ./...

.PHONY: coverage cover
cover: coverage
coverage:
	go test -cover -coverprofile cover.out ./... 

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: generate
generate:
	go generate ./...

.PHONY: show-coverage
show-coverage: coverage
	go tool cover -html cover.out

.PHONY: test
test:
	go test -race ./...

.PHONY: install
install:
	go install ./

GOOS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH ?= $(shell uname -m)

dist/garotate-$(GOOS)-$(GOARCH):
	go build -o dist/garotate-$(GOOS)-$(GOARCH) ./

.PHONY: release-binary
release-binary: dist/garotate-$(GOOS)-$(GOARCH)

.PHONY: clean
clean:
	rm -rf cover.out
	rm -rf dist/*

