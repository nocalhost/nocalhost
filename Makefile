SHELL := /bin/bash
BASEDIR = $(shell pwd)

.PHONY: api-docs
api-docs:
	@swag init -g cmd/nocalhost-api/nocalhost-api.go
	@echo "gen-docs done"
	@echo "see docs by: http://localhost:8080/swagger/index.html"

.PHONY: api
api: ## Build the binary file
	@go build cmd/nocalhost-api/nocalhost-api.go

.PHONY: gotool
gotool:
	gofmt -w .
	go tool vet . | grep -v vendor;true

.PHONY: dep
dep: ## Get the dependencies
	@go mod download

.PHONY: help
help:
	@echo "make [module ex: api] - compile the source code"
	@echo "make clean - remove binary file and vim swp files"
	@echo "make gotool - run go tool 'fmt' and 'vet'"
	@echo "make ca - generate ca files"
	@echo "make gen-docs - gen swag doc"