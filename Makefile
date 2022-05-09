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

TARGET_OS ?= $(HOST_OS)
TARGET_ARCH ?= $(HOST_ARCH)

ifeq ($(HOST_ARCH), x86_64)
	HOST_ARCH = amd64
endif
ifeq ($(TARGET_ARCH), x86_64)
	TARGET_ARCH = amd64
endif

BINARY_EXT =
ifeq ($(TARGET_OS), windows)
	BINARY_EXT = .exe
endif

GOOS ?= $(TARGET_OS)
GOARCH ?= $(TARGET_ARCH)

dist/garotate-$(GOOS)-$(GOARCH)$(BINARY_EXT):
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ ./

dist/garotate-darwin-universal-arm64: 
	make dist/garotate-darwin-arm64 GOOS=darwin GOARCH=arm64
	cp dist/garotate-$(GOOS)-$(GOARCH) dist/garotate-darwin-universal-arm64

dist/garotate-darwin-universal-amd64: 
	make dist/garotate-darwin-amd64 GOOS=darwin GOARCH=amd64
	cp dist/garotate-$(GOOS)-$(GOARCH) dist/garotate-darwin-universal-amd64

dist/garotate-darwin-universal: dist/garotate-darwin-universal-amd64 dist/garotate-darwin-universal-arm64
	lipo -create -output dist/garotate-darwin-universal $<

.PHONY: release-binary
release-binary: dist/garotate-$(GOOS)-$(GOARCH)$(BINARY_EXT)

S3BASEURL := s3://garotate.qubling.cloud
S3BUCKET ?= releases
S3URL = $(S3BASEURL)/$(S3BUCKET)

.PHONY: upload-release-binary upload-release-binary-universal
upload-release-binary: release-binary
	aws s3 cp dist/garotate-$(GOOS)-$(GOARCH)$(BINARY_EXT) $(S3URL)/garotate-$(GOOS)-$(GOARCH)$(BINARY_EXT)

upload-release-binary-universal: dist/garotate-darwin-universal
	aws s3 cp dist/garotate-darwin-amd64 $(S3URL)/garotate-darwin-amd64
	aws s3 cp dist/garotate-darwin-arm64 $(S3URL)/garotate-darwin-arm64
	aws s3 cp dist/garotate-darwin-universal $(S3URL)/garotate-darwin-universal

.PHONY: begin-release finalize-release
begin-release:
	./scripts/start-release $(VERSION)

finalize-release:
	./scripts/finish-release \
		garotate-darwin-amd64 \
		garotate-darwin-arm64 \
		garotate-darwin-universal \
		garotate-linux-amd64 \
		garotate-linux-arm64 \
		garotate-windows-386.exe \
		garotate-windows-amd64.exe \
		garotate-windows-arm64.exe

.PHONY: clean
clean:
	rm -rf cover.out
	rm -rf dist/*

