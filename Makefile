.PHONY: build test clean

APP         = gina
VERSION     = $(shell git describe --tags --abbrev=0)
GO          = go
GO_BUILD    = $(GO) build
GO_TEST     = $(GO) test -v
GO_TOOL     = $(GO) tool
GOOS        = ""
GOARCH      = ""
GO_PKGROOT  = ./...
GO_PACKAGES = $(shell $(GO_LIST) $(GO_PKGROOT))
GINA_SRCS = $(shell find cmd/gina -name "*.go"  -not -name '*_test.go')

build: ## Build project
	cd cmd/gina && $(GO_BUILD) -o ../../$(APP) ./...

clean: ## Clean project
	-rm -rf cover.out cover.html $(APP)

test: ## Start imaging package test
	env GOOS=$(GOOS) $(GO_TEST) -cover $(GO_PKGROOT) -coverprofile=cover.out
	$(GO_TOOL) cover -html=cover.out -o cover.html

test-gina: ## start gina package test
	env GOOS=$(GOOS) $(GO_TEST) -cover ./cmd/gina/... -coverprofile=cover.out
	$(GO_TOOL) cover -html=cover.out -o cover.html

bench: ## Start benchmark
	env GOOS=$(GOOS) $(GO_TEST) -bench .

compare-bench: ## Start compare benchmark between current and original code
	cob --threshold 0.1 --base "d471645c770227ca1e63837f1a52cb647e30a11a"

.DEFAULT_GOAL := help
help:  
	@grep -E '^[0-9a-zA-Z_-]+[[:blank:]]*:.*?## .*$$' $(MAKEFILE_LIST) | sort \
	| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[1;32m%-15s\033[0m %s\n", $$1, $$2}'