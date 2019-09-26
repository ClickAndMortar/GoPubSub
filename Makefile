all: build

default: build

build:
	GOOS=linux GOARCH=amd64 go build -o ./bin/gopubsub-linux main.go
	GOOS=darwin GOARCH=amd64 go build -o ./bin/gopubsub-darwin main.go

docker:
	docker build -t clickandmortar/gopubsub .
