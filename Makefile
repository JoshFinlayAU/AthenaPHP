# Athena PHP — encoder (Go) + decoder extension (C, built in Debian 13 container)

KEY        ?= athena.key
HEADER     := ext/athena/athena_key.h
GO         ?= go
DOCKER     := build/docker.sh
PKG_VERSION ?= 0.1.0

.PHONY: all build test key ext ext-local deb itest image encode clean distclean help

all: build ## Build the Go encoder (default)

build: ## Build the Go encoder -> bin/athena
	$(GO) build -o bin/athena ./cmd/athena

test: ## Run Go unit tests
	$(GO) test ./...

$(KEY): | build
	./bin/athena keygen -key $(KEY) -header $(HEADER)

key: $(KEY) ## Generate project key + embedded C header (once)

$(HEADER): $(KEY) | build
	./bin/athena header -key $(KEY) -out $(HEADER)

image: ## Build the Debian 13 build image
	docker build -t athena-build:deb13 -f build/Dockerfile.debian13 build

ext: $(HEADER) ## Build athena.so for PHP 8.3 & 8.4 (Docker)
	$(DOCKER) build/build-ext.sh

ext-local: $(HEADER) ## Build athena.so against the local PHP (no Docker)
	build/build-ext-local.sh

deb: ext ## Build the .deb package (Docker)
	$(DOCKER) "PKG_VERSION=$(PKG_VERSION) build/package-deb.sh"

itest: ext build ## Encode fixtures and run integration tests under 8.3 & 8.4 (Docker)
	rm -rf build/test-encoded
	./bin/athena encode -key $(KEY) -out build/test-encoded -quiet test/fixtures/src
	$(DOCKER) build/test-ext.sh

encode: build ## Encode a project: make encode SRC=path [OUT=dir]
	@test -n "$(SRC)" || { echo "usage: make encode SRC=path [OUT=dir]"; exit 2; }
	./bin/athena encode -key $(KEY) $(if $(OUT),-out $(OUT)) $(SRC)

clean: ## Remove Go build output and intermediate artifacts
	rm -rf bin build/out build/test-encoded build/*.deb

distclean: clean ## Also remove the secret key + generated header
	rm -f $(KEY) $(HEADER)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
