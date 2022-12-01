VERSION := $(shell git describe --tags $(shell git rev-list --tags --max-count=1))-next
COMMIT := $(shell git rev-parse --short HEAD)
BUILDFLAGS = -ldflags "-X github.com/linuxsuren/cobra-extension/version.version=$(VERSION) \
	-X github.com/linuxsuren/cobra-extension/version.commit=$(COMMIT) \
	-X github.com/linuxsuren/cobra-extension/version.date=$(shell date +'%Y-%m-%d') -w -s"

build:
	GO111MODULE=on CGO_ENABLE=0 GOARCH=amd64 GOOS=$(shell go env GOOS) go build $(BUILDFLAGS) -o bin/mp main.go
	chmod u+x bin/mp

copy: build
	sudo cp bin/mp /usr/local/bin/mp

pre-commit:
	hd i act
	act -j build
