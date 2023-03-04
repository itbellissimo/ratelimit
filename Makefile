BINARY_NAME=rate-limit

build:
	GOARCH=amd64 GOOS=darwin go build -o ./bin/server/${BINARY_NAME}-darwin ./cmd/server/server.go && \
 	GOARCH=amd64 GOOS=linux go build -o ./bin/server/${BINARY_NAME}-linux ./cmd/server/server.go && \
 	GOARCH=amd64 GOOS=windows go build -o ./bin/server/${BINARY_NAME}-windows.exe ./cmd/server/server.go

run-linux: build
	./bin/server/${BINARY_NAME}-linux

tidy:
	go mod tidy
.PHONY: tidy

vendor:
	go mod vendor

clean:
	go clean
	rm ${BINARY_NAME}-darwin
	rm ${BINARY_NAME}-linux
	rm ${BINARY_NAME}-windows

test:
	go test ./...

test-race:
	go clean -testcache && go test ./... -race

_clean-test-cache:
	go clean -testcache

test-without-cache: _clean-test-cache test
	#go test ./...
	#go test -count=1
	#GOCACHE=off go test

test_coverage:
	go clean -testcache && go test ./... -race -coverprofile=coverage.out

test_coverage-html: test_coverage
	go tool cover -html=coverage.out

dep:
	go mod download

vet:
	go vet

lint:
	golangci-lint run --enable-all

mod-download: ## Downloads the Go module.
	@echo "==> Downloading Go module"
	go mod download
.PHONY: mod-download
