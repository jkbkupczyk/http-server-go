SHELL = /bin/sh

run: build
	@./bin/go-httpserver $(ARGS)

build:
	@go build -o bin/go-httpserver ./app/.

test:
	@go test -v -timeout 30s ./...
