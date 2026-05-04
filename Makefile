.PHONY: install build

build:
	go build -o pwf ./cmd/pwf
	@echo "Built ./pwf"

install:
	go install ./cmd/pwf
	@echo "Installed pwf to $(shell go env GOPATH)/bin/pwf"
