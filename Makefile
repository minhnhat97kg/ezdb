.PHONY: build run test clean

build:
	CGO_ENABLED=1 go build -o bin/ezdb ./cmd/ezdb

run: build
	./bin/ezdb

test:
	CGO_ENABLED=1 go test ./...

clean:
	rm -rf bin/
