.PHONY: build test lint clean

BINARY := scry

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) .

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
