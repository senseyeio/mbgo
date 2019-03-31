GOPACKAGES := $(shell go list ./...)

.PHONY: errcheck
errcheck:
	@errcheck -asserts -blank -ignore 'io:[cC]lose' $(GOPACKAGES)

.PHONY: fmt
fmt:
	@go fmt $(PACKAGES)

.PHONY: integration
integration:
	@go test -cover -tags=integration -timeout=5s $(GOPACKAGES)

.PHONY: lint
lint:
	@golint -set_exit_status $(GOPACKAGES)

.PHONY: tools
tools:
	@go get -u golang.org/x/lint/golint github.com/kisielk/errcheck

.PHONY: unit
unit:
	@go test -cover -timeout=1s $(GOPACKAGES)