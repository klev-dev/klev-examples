.PHONY: update-api all

update-api:
	go get github.com/klev-dev/klev-api-go@main
	go mod tidy

all:
	go build -v -o bin/ ./...
