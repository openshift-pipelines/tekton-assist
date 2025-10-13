MODULE   = $(shell env GO111MODULE=on $(GO) list -m)
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || \
			cat $(CURDIR)/.version 2> /dev/null || echo v0)
PKGS     = $(or $(PKG),$(shell env GO111MODULE=on $(GO) list ./...))
TESTPKGS = $(shell env GO111MODULE=on $(GO) list -f \
			'{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' \
			$(PKGS))
BIN      = $(CURDIR)/.bin

GOLANGCI_VERSION = v1.47.2

GO           = go
TIMEOUT_UNIT = 5m
TIMEOUT_E2E  = 20m
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m🐱\033[0m")

export GO111MODULE=on

# Default KO_DOCKER_REPO if not provided
export KO_DOCKER_REPO ?= kind.local

COMMANDS=$(patsubst cmd/%,%,$(wildcard cmd/*))
BINARIES=$(addprefix bin/,$(COMMANDS))

.PHONY: all
all: fmt $(BINARIES) | $(BIN) ; $(info $(M) building executable…) @ ## Build program binary

$(BIN):
	@mkdir -p $@
$(BIN)/%: | $(BIN) ; $(info $(M) building $(PACKAGE)…)
	$Q tmp=$$(mktemp -d); cd $$tmp; \
		env GO111MODULE=on GOPATH=$$tmp GOBIN=$(BIN) $(GO) install $(PACKAGE) \
		|| ret=$$?; \
		env GO111MODULE=on GOPATH=$$tmp GOBIN=$(BIN) $(GO) clean -modcache \
        || ret=$$?; \
		cd - ; \
	  	rm -rf $$tmp ; exit $$ret

FORCE:

bin/%: cmd/% FORCE
	@mkdir -p $(dir $@)
	$Q $(GO) build -mod=vendor $(LDFLAGS) -v -o $@ ./$<

KO = $(or ${KO_BIN},${KO_BIN},$(BIN)/ko)
$(BIN)/ko: PACKAGE=github.com/google/ko@latest

KUSTOMIZE = $(or ${KUSTOMIZE_BIN},kustomize)
KUBECTL = $(or ${KUBECTL_BIN},kubectl)

# ko image build/publish

# Import path to build as the container image (ko publishes this)
IMAGE_IMPORT_PATH ?= ./cmd/tkn-assist

# Platforms to build for (comma-separated), e.g. linux/amd64,linux/arm64
PLATFORMS ?= linux/amd64

.PHONY: image
image: | $(KO) ; $(info $(M) building container image with ko…) @ ## Build/publish container image using ko
	$Q CGO_ENABLED=0 $(KO) build $(IMAGE_IMPORT_PATH) --platform=$(PLATFORMS) --tags $(VERSION)

.PHONY: build
build: bin/tkn-assist ; $(info $(M) building tkn-assist binary…) @ ## Build the tkn-assist CLI binary

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning…) 	@ ## Cleanup everything
	@rm -rf $(BIN)
	@rm -rf bin
	@rm -rf test/tests.* test/coverage.*

.PHONY: help
help:
	@grep -hE '^[ a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:
	@echo $(VERSION)


