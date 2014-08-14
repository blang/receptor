TAGS=plugins_all

build:
	@go get ./...
	@go build -o "receptor" -tags $(TAGS) ./cli

test:
	go test ./...

.PHONY: build