
# test packages
TEST_PKGS := perc/...

.PHONY: all test

all: local

test: ## Run tests
	@if [ ! -z "$(TEST_PKGS)" ]; then go test -test.v $(TEST_PKGS); fi
