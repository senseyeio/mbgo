GOPACKAGES := $(shell go list ./...)

.PHONY: default
default: fmt lint unit

.PHONY: errcheck
errcheck:
	@errcheck -asserts -blank -ignore 'io:[cC]lose' $(GOPACKAGES)

.PHONY: fmt
fmt:
	@go fmt $(PACKAGES)

.PHONY: integration
integration:
	@sh ./scripts/integration_test.sh $(GOPACKAGES)

.PHONY: lint
lint:
	@golint -set_exit_status $(GOPACKAGES)

.PHONY: unit
unit:
	@go test -cover -timeout=1s $(GOPACKAGES)