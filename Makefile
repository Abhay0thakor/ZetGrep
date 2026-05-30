BINARY_NAME=zetgrep
MAIN_PATH=./cmd/zetgrep

.PHONY: all build clean test bench run

all: test build

build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

clean:
	rm -f $(BINARY_NAME)
	rm -rf test_env/data/sample.txt*
	rm -f report.html out.json out.json.zst out.txt.zst results.json.zst report.txt.zst

test:
	go test -v ./...

bench:
	go test -bench=. ./pkg/scanner/...

run: build
	./$(BINARY_NAME)
