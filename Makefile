.PHONY: update-api all

all:
	go build -v -o bin/ ./...

update-api:
	go get github.com/klev-dev/klev-api-go@main
	go mod tidy
