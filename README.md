# mbgo

[![GoDoc](https://godoc.org/github.com/senseyeio/mbgo?status.svg)](https://godoc.org/github.com/senseyeio/mbgo) [![Build Status](https://travis-ci.org/senseyeio/mbgo.svg?branch=master)](https://travis-ci.org/senseyeio/mbgo)

A mountebank API client for the Go programming language.

## Installation

```sh
$ go get -u github.com/senseyeio/mbgo
```

## Testing

This package includes both unit and integration tests, with the integration tests currently tested against a mountebank v1.14.0 instance.

Both types of tests are run by default, but the integration tests can be skipped by testing in short mode:

```sh
$ go test -short
```

Otherwise, the integration tests' client points to a local Docker container at port 2525, with the additional ports 8080-8083 exposed for communication with the imposter fixtures.

## Contributing

* Fork the repository.
* Code your changes.
* If applicable, add tests and/or documentation.
* Please ensure all tests pass and that all code passes `golint`, `go vet` and `errcheck` (see the `Makefile` for more details).
* Raise a new pull request with a short description of your changes.
