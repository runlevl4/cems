SHELL := /bin/bash

run:
	go run main.go

build:
	go build main.go

tidy:
	go mod vendor
	go mod tidy

linux-service:
	sudo cp linux/cems.service /etc/systemd/system
	sudo chmod +x /etc/systemd/system/cems.service
