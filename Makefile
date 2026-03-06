SHELL := /bin/bash

.PHONY: default clean build fmt lint lint-ci shellcheck abcgo style run test cover integration_tests license before_commit help godoc install_docgo install_addlicense gen_mocks install_mockgen install_golangci-lint

SOURCES:=$(shell find . -name '*.go')
BINARY:=parquet-factory
DOCFILES:=$(addprefix docs/packages/, $(addsuffix .html, $(basename ${SOURCES})))

default: build

clean: ## Run go clean
	go clean
	rm -f ${BINARY}

build: ${BINARY} ## Keep this rule for compatibility

${BINARY}: ${SOURCES}
	./build.sh

install_golangci-lint:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

fmt: install_golangci-lint ## Run go formatting
	@echo "Running go formatting"
	golangci-lint fmt

lint: install_golangci-lint gen_mocks ## Run go liting
	@echo "Running go linting"
	golangci-lint run --fix

lint-ci: install_golangci-lint gen_mocks ## Run go liting without fixing
	@echo "Running go linting"
	golangci-lint run --timeout=3m

shellcheck: ## Run shellcheck
	./shellcheck.sh

abcgo: ## Run ABC metrics checker
	@echo "Run ABC metrics checker"
	./abcgo.sh ${VERBOSE}

unit_tests: clean gen_mocks build  ## Run the unit tests
	@echo "Running unit tests"
	./unit-tests.sh

coverage.out: unit_tests ## Run unit tests when coverage.out is not created

cover: coverage.out ## Check coverage of tests. Require generated file coverage.out from unit-tests.sh
	@./check_coverage.sh

style: fmt lint shellcheck ## Run all the formatting related commands (fmt, lint) + check shell scripts

run: ${BINARY} ## Build the project and executes the binary
	./$^

integration_tests: build ## Run BDD integration tests locally (requires docker-compose up)
	@echo "Running BDD integration tests"
	@behave tests/bdd/features --no-capture --format pretty

license: install_addlicense
	addlicense -c "Red Hat, Inc" -l "apache" -v ./

test: unit_tests integration_tests ## Run unit tests and integration tests

before_commit: style test license ## Checks done before commit
	./check_coverage.sh

help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''

docs/packages/%.html: %.go
	mkdir -p $(dir $@)
	docgo -outdir $(dir $@) $^
	addlicense -c "Red Hat, Inc" -l "apache" -v $@

install_addlicense:
	[[ `command -v addlicense` ]] || go install github.com/google/addlicense@latest

gen_mocks: install_mockgen s3writer/mock/s3writer.go
install_mockgen:
	[[ `command -v mockgen` ]] || go install github.com/golang/mock/mockgen@latest

s3writer/mock/s3writer.go: s3writer/types.go
	mkdir -p `dirname $@`
	mockgen -source $< -package mock > $@

