.PHONY: all
all: generate fmt test analyze

.PHONY: analyze
anaylyze:
	golangci-lint run ./...

.PHONY: clean
clean:
	rm -rf cover.out

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

garotate-$(GOOS)-$(GOARCH):
	go build -o garotate-$(GOOS)-$(GOARCH) ./

.PHONY: release-packages
release-packages: garotate-$(GOOS)-$(GOARCH)
