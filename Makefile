GOPACKAGES = $(shell go list ./... | grep -v -e '.*[/-]vendor.*')

.PHONY: default errcheck fmt lint test testshort tools vet

default: errcheck fmt lint test vet

errcheck:
	errcheck -asserts $(GOPACKAGES)

fmt:
	@for pkg in $(GOPACKAGES); do go fmt -x $$pkg; done

lint:
	golint -set_exit_status $(GOPACKAGES)

test:
	go test -cover $(GOPACKAGES)

testshort:
	go test -short $(GOPACKAGES)

tools:
	go get -u github.com/golang/lint/golint
	go get -u github.com/kisielk/errcheck

vet:
	go vet $(GOPACKAGES)
