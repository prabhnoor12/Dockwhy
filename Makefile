.PHONY: build clean lint test vet

build:
	go build ./...

clean:
	rm -f coverage.out

lint:
	golangci-lint run ./...

test:
	go test -race ./...

test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

vet:
	go vet ./...
