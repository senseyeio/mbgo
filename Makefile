PACKAGES := $(shell go list ./... | grep -v mock)

.PHONY: default
default: fmt lint

.PHONY: fmt
## fmt: runs go fmt on source files
fmt:
	@go fmt $(PACKAGES)

.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: integration
## integration: runs the integration tests
integration:
	@go clean -testcache
	@sh ./scripts/integration_test.sh $(PACKAGES)

.PHONY: lint
## lint: runs go lint on source files
lint:
	@golint -set_exit_status -min_confidence=0.3 $(PACKAGES)

.PHONY: unit
## unit: runs the unit tests
unit:
	@go clean -testcache
	@go test -cover -covermode=atomic -race -timeout=1s $(PACKAGES)