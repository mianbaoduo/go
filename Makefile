SHA := $(shell git rev-parse HEAD)

.PHONY: all clean develop publish

all: bin/go

bin/go: cmd/go/main.go $(shell find internal -name '*.go')
	go build -o $@ ./cmd/go

bin/devserver: cmd/devserver/main.go
	go build -o $@ ./cmd/devserver

develop: bin/devserver bin/go
	bin/devserver

clean:
	rm -rf bin

bin/buildimg:
	GOBIN="$(CURDIR)/bin" go install github.com/kellegous/buildimg@latest

bin/publish: cmd/publish/main.go
	go build -o $@ ./cmd/publish

publish: bin/publish
	bin/publish \
		--tag=latest \
		--tag=$(shell git rev-parse --short $(SHA)) \
		--platform=linux/arm64 \
		--platform=linux/amd64 \
		--image=kellegous/go
