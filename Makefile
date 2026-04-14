.PHONY: build test lint clean install

BINARY_NAME=zetgrep

build:
	go build -o $(BINARY_NAME) ./cmd/zetgrep

test:
	go test -v ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY_NAME)
	rm -rf results/*.json

install:
	go install ./cmd/zetgrep
