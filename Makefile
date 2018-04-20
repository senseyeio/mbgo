PACKAGE = github.com/senseyeio/mbgo
GOPACKAGES = $(shell go list ./... | grep -v -e '.*[/-]mock.*')

.PHONY: default errcheck fmt lint test tools vet

default: errcheck fmt lint test vet

errcheck:
	@for pkg in $(GOPACKAGES); do errcheck -asserts $$pkg; done

fmt:
	@for pkg in $(GOPACKAGES); do go fmt $$pkg; done

lint:
	@for pkg in $(GOPACKAGES); do golint $$pkg; done

test:
	@for pkg in $(GOPACKAGES); do go test -v -covermode=atomic -coverprofile=coverage.txt -race $$pkg; done

testshort:
	@for pkg in $(GOPACKAGES); do go test -short $$pkg; done

tools:
	go get -u github.com/golang/lint/golint
	go get -u github.com/kisielk/errcheck

vet:
	@for pkg in $(GOPACKAGES); do go vet $$pkg; done
