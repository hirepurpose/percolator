
# path to the project root
export PROJECT	:= $(shell cd $(PWD)/.. && pwd)
export VENDOR		:= $(PROJECT)/vendor

# main project root
COMPONENT := percolator
# the component's main package
MAIN := ./src/perc/main

# sources
SRC	= $(shell find src -name \*.go -print)
LIB	= $(shell find $(FRAMEWORK)/src/hp $(VENDOR)/src/github.com/bww -name \*.go -print)

# include the base makefile
include ../build/subproj.make
# include the environment
include ../build/env/$(ENVIRON).make

# environment
export HP_HOME := $(BUILD_DIR)

# the version to tag images with
VERSION			= latest
# the remote image repo
IMAGE_REPO	= 371393096049.dkr.ecr.us-east-1.amazonaws.com/hirepurpose/$(PRODUCT_NAME)
# the local image tag
LOCAL_TAG 	= hirepurpose/$(PRODUCT_NAME):latest
# the remote image tag
REMOTE_TAG 	= $(IMAGE_REPO):$(VERSION)

# test packages
TEST_PKGS := perc/service

.PHONY: all web run test stage release

all: local

run: local ## Build and run the service with default parameters
	$(PRODUCT) -debug $(FLAGS)

test: ## Run tests
	@if [ ! -z "$(TEST_PKGS)" ]; then go test -test.v $(TEST_PKGS); fi

stage: export EXPECT_BRANCH ?= staging
stage: export DEPLOY_CLUSTER = Sandbox
stage: export DEPLOY_TASK = SandboxPercolator
stage: export DEPLOY_SERVICE = SandboxPercolator
stage: export MIN_PERCENT_DEPLOYMENT = 0
stage: export MAX_PERCENT_DEPLOYMENT = 100
stage: export ENVIRON = staging
stage: export VERSION = staging
stage: clean deploy ## Build and push an updated image to Elastic Container Service and deploy the update on the staging cluster

release: export EXPECT_BRANCH ?= master
release: export DEPLOY_CLUSTER = Discovery
release: export DEPLOY_TASK = Percolator
release: export DEPLOY_SERVICE = Percolator
release: export MIN_PERCENT_DEPLOYMENT = 50
release: export MAX_PERCENT_DEPLOYMENT = 200
release: export ENVIRON = release
release: export VERSION = production
release: clean deploy ## Build and push an updated image to Elastic Container Service and deploy the update on the production cluster
