SHELL := /bin/bash

run:
    go run main.go

build:
    go build main.go

tidy:
    go mod vendor
    go mod tidy
