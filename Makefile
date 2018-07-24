GOPACKAGES := $(shell go list github.com/senseyeio/mbgo/...)

.PHONY: fmt
fmt:
	@for pkg in $(GOPACKAGES); do go fmt $$pkg; done

.PHONY: lint
lint:
	@golint -set_exit_status $(GOPACKAGES)

.PHONY: unit
unit:
	@go test -cover -short $(GOPACKAGES)

.PHONY: integration
integration:
	@go test -cover -cpu=1,2 -tags integration $(GOPACKAGES)

.PHONY: tools
tools:
	@go get -u golang.org/x/lint/golint
