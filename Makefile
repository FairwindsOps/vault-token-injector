# Go parameters
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=vault-token-injector
COMMIT := $(shell git rev-parse HEAD)
VERSION := "local-dev"

all: lint test
build:
	$(GOBUILD) -o $(BINARY_NAME) -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -s -w" -v
lint:
	golangci-lint run
reportcard:
	goreportcard-cli -t 100 -v
test:
	GO111MODULE=on $(GOCMD) test -v --bench --benchmem -coverprofile coverage.txt -covermode=atomic ./...
	GO111MODULE=on $(GOCMD) vet ./... 2> govet-report.out
	GO111MODULE=on $(GOCMD) tool cover -html=coverage.txt -o cover-report.html
	printf "\nCoverage report available at cover-report.html\n\n"
tidy:
	$(GOCMD) mod tidy
clean:
	$(GOCLEAN)
	$(GOCMD) fmt ./...
	rm -f $(BINARY_NAME)
# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME) -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -s -w" -v
build-docker:
	docker build --build-arg version=$(VERSION) --build-arg commit=$(COMMIT) -t quay.io/fairwinds/$(BINARY_NAME):dev .
