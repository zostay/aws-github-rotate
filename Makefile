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

.PHONE: show-coverage
show-coverage: coverage
	go tool cover -html cover.out

.PHONY: test
test:
	go test -race ./...

install:
	go install ./
