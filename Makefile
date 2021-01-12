build:
	GO111MODULE=on CGO_ENABLE=0 GOARCH=amd64 GOOS=$(shell go env GOOS) go build -o bin/mp main.go
	chmod u+x bin/mp

copy: build
	sudo cp bin/mp /usr/local/bin/mp
