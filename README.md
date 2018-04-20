# mbgo

[![GoDoc](https://godoc.org/github.com/senseyeio/mbgo?status.svg)](https://godoc.org/github.com/senseyeio/mbgo) [![Build Status](https://travis-ci.org/senseyeio/mbgo.svg?branch=master)](https://travis-ci.org/senseyeio/mbgo)

A mountebank API client for the Go programming language.

```sh
$ go get -u github.com/senseyeio/mbgo
```

## Testing

This package includes both unit and integration tests, with integration tests being disabled in short mode:

```sh
$ go test -short
```

Otherwise the integration tests are ran against a local Docker container running mountebank v1.14.0 on port 2525, with additional ports 8080-8083 exposed for imposter fixtures.

## Examples

See the examples in the [godoc](https://godoc.org/github.com/senseyeio/mbgo).

## Contributing

* Fork the repository.
* Code your changes.
* If applicable, add tests and/or documentation.
* Please ensure all tests pass and that all code passes `golint`, `go vet` and `errcheck` (see the `Makefile` for more details).
* Raise a new pull request with a short description of your changes.
