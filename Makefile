CGO_ENABLED := 1
NAME := sporeprint
ICON := "🧿"
ORG := github.com/farcloser

MAKEFILE_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
VERSION ?= $(shell git -C $(MAKEFILE_DIR) describe --match 'v[0-9]*' --dirty='.m' --always --tags 2>/dev/null \
	|| echo "no_git_information")
VERSION_TRIMMED := $(VERSION:v%=%)
COMMIT ?= $(shell git -C $(MAKEFILE_DIR) rev-parse HEAD 2>/dev/null || echo "no_git_information")$(shell \
	if ! git -C $(MAKEFILE_DIR) diff --no-ext-diff --quiet --exit-code 2>/dev/null; then echo .m; fi)
LINT_COMMIT_RANGE ?= main..HEAD
DATE = "$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

ifdef VERBOSE
	VERBOSE_FLAG := -v
	VERBOSE_FLAG_LONG := --verbose
endif

ifndef NO_COLOR
    NC := \033[0m
    GREEN := \033[1;32m
    ORANGE := \033[1;33m
    BLUE := \033[1;34m
    RED := \033[1;31m
endif

# Helpers
recursive_wildcard=$(wildcard $1$2) $(foreach e,$(wildcard $1*),$(call recursive_wildcard,$e/,$2))

define title
	@printf "$(GREEN)____________________________________________________________________________________________________\n" 1>&2
	@printf "$(GREEN)%*s\n" $$(( ( $(shell echo "$(ICON)$(1) $(ICON)" | wc -c ) + 100 ) / 2 )) "$(ICON)$(1) $(ICON)" 1>&2
	@printf "$(GREEN)____________________________________________________________________________________________________\n$(ORANGE)" 1>&2
endef

define footer
	@printf "$(GREEN)> %s: done!\n" "$(1)" 1>&2
	@printf "$(GREEN)____________________________________________________________________________________________________\n$(NC)" 1>&2
endef

help:
	$(call title, $@)
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	$(call footer, $@)

# Tasks
lint: lint-go lint-commits lint-mod lint-licenses-all lint-headers lint-yaml lint-shell lint-go-all

fix: fix-go-all fix-mod ## Automatically fix some issues

test: unit ## Run all tests

unit: test-unit test-unit-race test-unit-bench ## Run unit tests

##########################
# Linting tasks
##########################
lint-go:
	$(call title, $@ $(GOOS))
	@cd $(MAKEFILE_DIR) \
		&& golangci-lint run $(VERBOSE_FLAG_LONG) ./...
	$(call footer, $@)

lint-go-all:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& GOOS=darwin make lint-go \
		&& GOOS=linux make lint-go \
		&& GOOS=freebsd make lint-go \
		&& GOOS=windows make lint-go
	$(call footer, $@)

lint-yaml:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& yamllint .
	$(call footer, $@)

lint-shell:
	$(call title, $@)
	@files=$$(find $(MAKEFILE_DIR) -name '*.sh' ! -path '*/tmp/*' ! -path '*/_*' 2>/dev/null); \
        if [ -n "$$files" ]; then shellcheck -a -x $$files; else echo "No shell scripts found, skipping shellcheck"; fi
	$(call footer, $@)

# See https://github.com/andyfeller/gh-ssh-allowed-signers for automation to retrieve contributors keys
lint-commits:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& { git config --unset-all gpg.ssh.allowedSignersFile hack/allowed_signers || true; } \
		&& git config --add gpg.ssh.allowedSignersFile hack/allowed_signers \
		&& git-validation $(VERBOSE_FLAG) -run DCO,short-subject,dangling-whitespace -range "$(LINT_COMMIT_RANGE)"
	$(call footer, $@)

lint-headers:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& ltag -t "./hack/headers" --check -v
	$(call footer, $@)

lint-mod:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go mod tidy --diff
	$(call footer, $@)

# FIXME: go-licenses cannot find LICENSE from root of repo when submodule is imported:
# https://github.com/google/go-licenses/issues/186
# This is impacting gotest.tools
lint-licenses:
	$(call title, $@: $(GOOS))
	@cd $(MAKEFILE_DIR) \
		&& go-licenses check --include_tests --allowed_licenses=Apache-2.0,BSD-2-Clause,BSD-3-Clause,MIT,MPL-2.0 \
		  --ignore gotest.tools \
		  --ignore github.com/in-toto \
		  --ignore github.com/cyberphone/json-canonicalization \
		  ./...
	$(call footer, $@)

lint-licenses-all:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& GOOS=darwin make lint-licenses \
		&& GOOS=linux make lint-licenses \
		&& GOOS=freebsd make lint-licenses \
		&& GOOS=windows make lint-licenses
	$(call footer, $@)

##########################
# Automated fixing tasks
##########################
fix-go:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& golangci-lint run --fix
	$(call footer, $@)

fix-go-all:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& GOOS=darwin make fix-go \
		&& GOOS=linux make fix-go \
		&& GOOS=freebsd make fix-go \
		&& GOOS=windows make fix-go
	$(call footer, $@)

fix-mod:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go mod tidy
	$(call footer, $@)

up:
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go get -u ./...
	$(call footer, $@)

##########################
# Development tools installation
##########################
install-dev-gotestsum:
	# gotestsum: 1.13.0 (2025-10-21)
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go install gotest.tools/gotestsum@c4a0df2e75a225d979a444342dd3db752b53619f
	$(call footer, $@)

install-dev-jsonschema:
	# go-jsonschema: v0.20.0 (omissis fork, 2025-06-19)
	$(call title, $@)
	@cd $(MAKEFILE_DIR) \
		&& go install github.com/atombender/go-jsonschema@v0.20.0
	$(call footer, $@)

install-dev-tools: install-dev-gotestsum install-dev-jsonschema
	$(call title, $@)
	# 2026-01-23
	# - golangci: v2.8.0
	# - git-validation: main
	# - ltag: main
	# - go-licenses: v2.0.1
	@cd $(MAKEFILE_DIR) \
		&& go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@e2e40021c9007020676c93680a36e3ab06c6cd33 \
		&& go install github.com/vbatts/git-validation@a8d455533459b620fa656bad095b943e70cede9b \
		&& go install github.com/containerd/ltag@66e6a514664ee2d11a470735519fa22b1a9eaabd \
		&& go install github.com/google/go-licenses/v2@3e084b0caf710f7bfead967567539214f598c0a2
	@echo "Remember to add \$$HOME/go/bin to your path"
	$(call footer, $@)

test-unit:
	$(call title, $@)
	@go test $(VERBOSE_FLAG) -count 1 $(MAKEFILE_DIR)/...
	$(call footer, $@)

test-unit-bench:
	$(call title, $@)
	@go test $(VERBOSE_FLAG) -count 1 $(MAKEFILE_DIR)/... -bench=.
	$(call footer, $@)

test-unit-race:
	$(call title, $@)
	@CGO_ENABLED=1 go test $(VERBOSE_FLAG) $(MAKEFILE_DIR)/... -race
	$(call footer, $@)

.PHONY: \
	lint \
	fix \
	test \
	up \
	unit \
	install-dev-tools install-dev-gotestsum install-dev-jsonschema \
	lint-commits lint-go lint-go-all lint-headers lint-licenses lint-licenses-all lint-mod lint-shell lint-yaml \
	fix-go fix-go-all fix-mod \
	test-unit test-unit-race test-unit-bench \
	build install clean

# Binary name
BINARY_PATH := ./bin/$(NAME)

UNAME_S := $(shell uname -s)

## https://gcc.gnu.org/onlinedocs/gcc/Warning-Options.html
WARNING_OPTIONS := -Wall -Werror=format-security
## https://gcc.gnu.org/onlinedocs/gcc/Optimize-Options.html#Optimize-Options
OPTIMIZATION_OPTIONS := -O2
OPTIMIZATION_OPTIONS_DEBUG := -O0
## https://gcc.gnu.org/onlinedocs/gcc/Debugging-Options.html#Debugging-Options
DEBUGGING_OPTIONS := -grecord-gcc-switches -g
## https://gcc.gnu.org/onlinedocs/gcc/Preprocessor-Options.html#Preprocessor-Options
# https://www.gnu.org/software/libc/manual/html_node/Source-Fortification.html
SECURITY_OPTIONS := -fstack-protector-strong -fPIE -D_FORTIFY_SOURCE=2
## Control flow integrity is amd64 only
# -mcet -fcf-protection

# C linker flags (passed to ld via CGO_LDFLAGS)
LDFLAGS_C :=
ifeq ($(UNAME_S),Linux)
    SECURITY_OPTIONS += -fstack-clash-protection
    LDFLAGS_C += -Wl,-z,defs -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack
endif

# C compiler flags
# -pipe gives a little speed-up by using pipes instead of temp files
CFLAGS := $(WARNING_OPTIONS) $(OPTIMIZATION_OPTIONS) $(SECURITY_OPTIONS) -pipe
CFLAGS_DEBUG := $(WARNING_OPTIONS) $(OPTIMIZATION_OPTIONS_DEBUG) $(DEBUGGING_OPTIONS) -D_GLIBCXX_ASSERTIONS -pipe
CPPFLAGS := -D_FORTIFY_SOURCE=2
CPPFLAGS_DEBUG := -D_GLIBCXX_ASSERTIONS
CXXFLAGS := $(CFLAGS)
CXXFLAGS_DEBUG := $(CFLAGS_DEBUG)

# Go linker flags
# -s strips symbol table, -w strips DWARF
LDFLAGS_VERSION := -X $(ORG)/$(NAME)/version.version=$(VERSION) \
    -X $(ORG)/$(NAME)/version.commit=$(COMMIT) \
    -X $(ORG)/$(NAME)/version.date=$(DATE)
LDFLAGS_BASE := -linkmode=external $(LDFLAGS_VERSION)
LDFLAGS_RELEASE := -s -w $(LDFLAGS_BASE) -extldflags='-pie'
LDFLAGS_DEBUG := $(LDFLAGS_BASE) -extldflags='-pie'
LDFLAGS_STATIC := -s -w $(LDFLAGS_BASE) -extldflags='-static'

# Go compiler flags
# -N disables optimizations, -l disables inlining
GCFLAGS_DEBUG := all=-N -l

# More reading:
## https://news.ycombinator.com/item?id=18874113
## https://developers.redhat.com/blog/2018/03/21/compiler-and-linker-flags-gcc
## https://gcc.gnu.org/onlinedocs/gcc/Instrumentation-Options.html
# https://github.com/golang/go/issues/26849

GOFLAGS := -tags=cgo,netgo,osusergo,static_build
export GOFLAGS

# Linker optimization  CGO_LDFLAGS=-fuse-ld=lld

GOCMD := go
GOBUILD := $(GOCMD) build -trimpath -buildmode=pie -ldflags '$(LDFLAGS_RELEASE)'
GOBUILD_DEBUG := $(GOCMD) build -buildmode=pie -gcflags='$(GCFLAGS_DEBUG)' -ldflags '$(LDFLAGS_DEBUG)'
GOBUILD_STATIC := $(GOCMD) build -trimpath -ldflags '$(LDFLAGS_STATIC)'

GOINSTALL := $(GOCMD) install

# Export CGO flags for release builds by default
export CGO_CFLAGS := $(CFLAGS)
export CGO_CPPFLAGS := $(CPPFLAGS)
export CGO_CXXFLAGS := $(CXXFLAGS)
export CGO_LDFLAGS := $(LDFLAGS_C)

build: ## Build the binary (PIE, release)
	@echo "Building $(NAME)..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/$(NAME)
	@echo "Binary built: $(BINARY_PATH)"

build-debug: ## Build the binary (PIE, debug)
build-debug: export CGO_CFLAGS := $(CFLAGS_DEBUG)
build-debug: export CGO_CPPFLAGS := $(CPPFLAGS_DEBUG)
build-debug: export CGO_CXXFLAGS := $(CXXFLAGS_DEBUG)
build-debug:
	@echo "Building $(NAME) (debug)..."
	@mkdir -p bin
	$(GOBUILD_DEBUG) -o $(BINARY_PATH)-debug ./cmd/$(NAME)
	@echo "Binary built: $(BINARY_PATH)"

build-static: ## Build static binary (Linux only, release)
	@echo "Building $(NAME) (static)..."
	@mkdir -p bin
	$(GOBUILD_STATIC) -o $(BINARY_PATH)-static ./cmd/$(NAME)
	@echo "Binary built: $(BINARY_PATH)"

## Binary name
#BINARY_PATH := ./bin/$(NAME)
#
#UNAME_S := $(shell uname -s)
#
### https://gcc.gnu.org/onlinedocs/gcc/Warning-Options.html
#WARNING_OPTIONS := -Wall -Werror=format-security
### https://gcc.gnu.org/onlinedocs/gcc/Optimize-Options.html#Optimize-Options
#OPTIMIZATION_OPTIONS := -O2
### https://gcc.gnu.org/onlinedocs/gcc/Debugging-Options.html#Debugging-Options
#DEBUGGING_OPTIONS := -grecord-gcc-switches -g
### https://gcc.gnu.org/onlinedocs/gcc/Preprocessor-Options.html#Preprocessor-Options
## https://www.gnu.org/software/libc/manual/html_node/Source-Fortification.html
#SECURITY_OPTIONS := -fstack-protector-strong -fPIE -D_FORTIFY_SOURCE=2 -D_GLIBCXX_ASSERTIONS
### Control flow integrity is amd64 only
## -mcet -fcf-protection
### https://gcc.gnu.org/onlinedocs/gcc/Link-Options.html#Link-Options
#
## C linker flags (passed to ld via CGO_LDFLAGS)
#LDFLAGS_C :=
#ifeq ($(UNAME_S),Linux)
#    SECURITY_OPTIONS += -fstack-clash-protection
#    LDFLAGS_C += -Wl,-z,defs -Wl,-z,relro -Wl,-z,now -Wl,-z,noexecstack
#endif
#
## Go linker flags
#LDFLAGS_VERSION := -X $(ORG)/$(NAME)/version.version=$(VERSION) \
#    -X $(ORG)/$(NAME)/version.commit=$(COMMIT) \
#    -X $(ORG)/$(NAME)/version.date=$(DATE)
## -s strips symbol table, -w strips DWARF
#LDFLAGS_BASE := -s -w -linkmode=external $(LDFLAGS_VERSION)
#LDFLAGS_NOPIE := $(LDFLAGS_BASE) -extldflags='-static'
#LDFLAGS_PIE := $(LDFLAGS_BASE) -extldflags='-pie'
#
## -pipe gives a little speed-up by using pipes instead of temp files
#CFLAGS := $(WARNING_OPTIONS) $(OPTIMIZATION_OPTIONS) $(SECURITY_OPTIONS) -pipe
#CPPFLAGS := -D_FORTIFY_SOURCE=2 -D_GLIBCXX_ASSERTIONS
#CXXFLAGS := $(CFLAGS)
#CGO_CFLAGS := $(CFLAGS)
#export CGO_CFLAGS
#CGO_CPPFLAGS := $(CPPFLAGS)
#export CGO_CPPFLAGS
#CGO_CXXFLAGS := $(CXXFLAGS)
#export CGO_CXXFLAGS
#CGO_LDFLAGS := $(LDFLAGS_C)
#export CGO_LDFLAGS
#
## More reading:
### https://news.ycombinator.com/item?id=18874113
### https://developers.redhat.com/blog/2018/03/21/compiler-and-linker-flags-gcc
### https://gcc.gnu.org/onlinedocs/gcc/Instrumentation-Options.html
## https://github.com/golang/go/issues/26849
#
#GOFLAGS := -tags=cgo,netgo,osusergo,static_build
#export GOFLAGS
#
## Linker optimization  CGO_LDFLAGS=-fuse-ld=lld
#
#GOCMD := go
#
#GOBUILD := $(GOCMD) build -trimpath -buildmode=pie -ldflags '$(LDFLAGS_PIE)'
#GOBUILD_STATIC := $(GOCMD) build -trimpath -ldflags '$(LDFLAGS_NOPIE)'
#
#GOINSTALL := $(GOCMD) install
#
#build: ## Build the binary (PIE)
#	@echo "Building $(NAME)..."
#	@mkdir -p bin
#	$(GOBUILD) -o $(BINARY_PATH) ./cmd/$(NAME)
#	@echo "Binary built: $(BINARY_PATH)"
#
#build-static: ## Build static binary (Linux only)
#	@echo "Building $(NAME) (static)..."
#	@mkdir -p bin
#	$(GOBUILD_STATIC) -o $(BINARY_PATH) ./cmd/$(NAME)
#	@echo "Binary built: $(BINARY_PATH)"
#
## TODO: -gcflags=all="-N -l"
#debug:


install: ## Install to GOPATH/bin
	@echo "Installing $(NAME)..."
	$(GOINSTALL) ./cmd/$(NAME)
	@echo "Installed to $$(go env GOPATH)/bin/$(NAME)"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/$(NAME)
	@echo "Clean complete"
