SHELL := /bin/bash
BASEDIR = $(shell pwd)

GIT_TAG := $(shell git describe 2>/dev/null | sed 's/refs\/tags\///' | sed 's/\(.*\)-.*/\1/' | sed 's/-[0-9]*$///' || true)
GIT_COMMIT_SHA := $(shell git rev-parse HEAD)
ifneq ($(shell git status --porcelain),)
    GIT_COMMIT_SHA := $(GIT_COMMIT_SHA)-dirty
endif

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
	@go build -ldflags '-X main.GIT_COMMIT_SHA=$(GIT_COMMIT_SHA)' cmd/nocalhost-api/nocalhost-api.go

.PHONY: nocalhost-dep
nocalhost-dep: ## Build nocalhost-dep
	@go build -ldflags '-X main.GIT_COMMIT_SHA=$(GIT_COMMIT_SHA)' cmd/nocalhost-dep/nocalhost-dep.go

.PHONY: nhctl
nhctl: ## Build nhctl
	@go build -ldflags '-X nocalhost/cmd/nhctl/cmds.GIT_COMMIT_SHA=$(GIT_COMMIT_SHA) -X nocalhost/cmd/nhctl/cmds.GIT_TAG=${GIT_TAG}' cmd/nhctl/nhctl.go

.PHONY: gotool
gotool: ## run go tool 'fmt' and 'vet'
	gofmt -w .
	go tool vet . | grep -v vendor;true

.PHONY: dep
dep: ## Get the dependencies
	@go mod download
