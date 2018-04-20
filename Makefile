PACKAGE = github.com/senseyeio/mbgo
GOPACKAGES = $(shell go list ./... | grep -v '.*[/-]vendor.*')

.PHONY: default errcheck fmt lint test testshort tools vet

default: errcheck fmt lint test vet

errcheck:
	errcheck -asserts -verbose $(GOPACKAGES)

fmt:
	@for pkg in $(GOPACKAGES); do go fmt -x $$pkg; done

lint:
	golint $(GOPACKAGES)

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic $(GOPACKAGES)

testshort:
	go test -v -short $(GOPACKAGES)

tools:
	go get -u github.com/golang/lint/golint
	go get -u github.com/kisielk/errcheck

vet:
	go vet $(GOPACKAGES)
