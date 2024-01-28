.PHONY: update-api all

all:
	go install -v ./...

update-api:
	go get github.com/klev-dev/klev-api-go@main
	go mod tidy
