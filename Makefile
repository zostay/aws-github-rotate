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

HOST_OS = $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_ARCH = $(shell uname -m)
TARGET_ARCH ?= $(HOST_ARCH)

ifeq ($(HOST_ARCH), x86_64)
	HOST_ARCH = amd64
endif
ifeq ($(TARGET_ARCH), x86_64)
	TARGET_ARCH = amd64
endif

GOOS ?= $(HOST_OS)
GOARCH ?= $(TARGET_ARCH)

dist/garotate-$(GOOS)-$(GOARCH):
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o dist/garotate-$(GOOS)-$(GOARCH) ./

dist/garotate-darwin-universal-arm64: 
	make dist/garotate-darwin-arm64 GOOS=darwin GOARCH=arm64
	cp dist/garotate-$(GOOS)-$(GOARCH) dist/garotate-darwin-universal-arm64

dist/garotate-darwin-universal-amd64: 
	make dist/garotate-darwin-amd64 GOOS=darwin GOARCH=amd64
	cp dist/garotate-$(GOOS)-$(GOARCH) dist/garotate-darwin-universal-amd64

dist/garotate-darwin-universal: dist/garotate-darwin-universal-amd64 dist/garotate-darwin-universal-arm64
	lipo -create -output dist/garotate-darwin-universal $<

.PHONY: release-binary
release-binary: dist/garotate-$(GOOS)-$(GOARCH)

.PHONY: upload-release-binary upload-release-binary-universal
upload-release-binary: release-binary
	aws s3 cp dist/garotate-$(GOOS)-$(GOARCH) $(S3URL)/garotate-$(GOOS)-$(GOARCH)

upload-release-binary-universal: dist/garotate-darwin-universal
	aws s3 cp dist/garotate-darwin-amd64 $(S3URL)/garotate-darwin-amd64
	aws s3 cp dist/garotate-darwin-arm64 $(S3URL)/garotate-darwin-arm64
	aws s3 cp dist/garotate-darwin-universal $(S3URL)/garotate-darwin-universal

.PHONY: clean
clean:
	rm -rf cover.out
	rm -rf dist/*

