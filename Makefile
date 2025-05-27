.PHONY: build run test clean

build:
	go build -o incidentio-alert-mcp .

run: build
	./incidentio-alert-mcp

test:
	go test -v ./...

clean:
	rm -f incidentio-alert-mcp
	go clean

fmt:
	go fmt ./...

lint:
	golangci-lint run

deps:
	go mod download
	go mod tidy