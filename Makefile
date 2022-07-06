SHELL := /bin/bash
BASEDIR = $(shell pwd)

.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: api-docs
api-docs: ## gen-docs - gen swag doc
	@swag init -g cmd/nocalhost-api/nocalhost-api.go
	@echo "gen-docs done"
	@echo "see docs by: http://localhost:8080/swagger/index.html"

.PHONY: api
api: ## Build nocalhost-api
	@bash ./scripts/build/api/build

.PHONY: api-docker
api-docker: ## Build nocalhost-api docker image
	@bash ./scripts/build/api/docker

.PHONY: dep-docker
dep-docker: ## Build nocalhost-dep docker image
	@bash ./scripts/build/dep/docker

.PHONY: dep-installer-job-docker
dep-installer-job-docker: ## Build dep-installer-job-docker docker image
	@bash ./scripts/build/dep/installer-job

.PHONY: vpn-docker
vpn-docker: nhctl-linux ## Build vpn docker image
	@bash ./scripts/build/vpn/docker

.PHONY: control-plane-docker
control-plane-docker:
	@bash ./scripts/build/control-plane/docker

.PHONY: envoy-docker
envoy-docker:
	@bash ./scripts/build/envoy/docker

.PHONY: nhctl
nhctl: ## Build nhctl for current OS
	@echo "WARNING: binary creates a current os executable."
	@bash ./scripts/build/nhctl/binary

.PHONY: nhctl-cross
nhctl-cross: ## build executable for Linux and macOS and Windows
	@bash ./scripts/build/nhctl/cross

.PHONY: nhctl-windows
nhctl-windows: ## build executable for Windows
	@bash ./scripts/build/nhctl/windows

.PHONY: nhctl-osx
nhctl-osx: ## build executable for macOS
	@bash ./scripts/build/nhctl/osx

.PHONY: nhctl-osx-arm64
nhctl-osx-arm64: ## build executable for M1
	@bash ./scripts/build/nhctl/osx-arm64

.PHONY: nhctl-linux
nhctl-linux: ## build executable for Linux
	@bash ./scripts/build/nhctl/linux

.PHONY: nhctl-linux-arm64
nhctl-linux-arm64: ## build executable for Linux arm64
	@bash ./scripts/build/nhctl/linux-arm64

.PHONY: gotool
gotool: ## run go tool 'fmt' and 'vet'
	gofmt -w .
	go tool vet . | grep -v vendor;true

.PHONY: dep
dep: ## Get the dependencies
	@go mod download

clean: ### Remove build dir
	@rm -fr build
