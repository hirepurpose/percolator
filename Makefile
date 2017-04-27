
# environment
export ENVIRON ?= devel

# update the gopath
GOPATH := $(PWD)

# default environment
GOOS		?= $(shell go env GOOS)
GOARCH	?= $(shell go env GOARCH)

# build paths
ARCH				= $(GOOS)_$(GOARCH)
OUTPUT_DIR	= $(PWD)/target
ARCH_DIR		=	$(OUTPUT_DIR)/$(ARCH)
TARGET_DIR	= $(ARCH_DIR)/$(ENVIRON)
BUILD_DIR		= $(TARGET_DIR)/product

# main project root
COMPONENT := percolator
# the component's main package
MAIN := ./src/perc/main
# product name
PRODUCT_NAME = $(COMPONENT)
# build and packaging
PRODUCT	= $(BUILD_DIR)/bin/$(PRODUCT_NAME)

# sources
SRC	= $(shell find src -name \*.go -print)

# test packages
# TEST_PKGS := $(COMPONENT)/service

.PHONY: all run test build

all: build

run: build ## Build and run the service with default parameters
	$(PRODUCT) -debug $(FLAGS)

test: ## Run tests
	@if [ ! -z "$(TEST_PKGS)" ]; then go test -test.v $(TEST_PKGS); fi

$(PRODUCT): $(SRC)
	mkdir -p $(BUILD_DIR)/bin && go build -o $(PRODUCT) $(MAIN)

build: $(PRODUCT) ## Build the product
