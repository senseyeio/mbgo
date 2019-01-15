# mbgo

[![GoDoc](https://godoc.org/github.com/senseyeio/mbgo?status.svg)](https://godoc.org/github.com/senseyeio/mbgo) [![Build Status](https://travis-ci.org/senseyeio/mbgo.svg?branch=master)](https://travis-ci.org/senseyeio/mbgo) [![Go Report Card](https://goreportcard.com/badge/github.com/senseyeio/mbgo)](https://goreportcard.com/report/github.com/senseyeio/mbgo)

A mountebank API client for the Go programming language.

## Installation

```sh
$ go get -u github.com/senseyeio/mbgo
```

## Testing

This package includes both unit and integration tests. Use the `unit` and `integration` targets in the Makefile to run them, respectively:

```sh
$ make unit
$ make integration
```

The integration test client points to a local Docker container at port 2525, with the additional ports 8080-8083 exposed for communication with test imposters. Currently tested against a mountebank v1.16.0 instance.

## Contributing

* Fork the repository.
* Code your changes.
* If applicable, add tests and/or documentation.
* Please ensure all unit and integration tests are passing, and that all code passes `make lint`.
* Raise a new pull request with a short description of your changes.
* Use the following convention for branch naming: `<username>/<description-with-dashes>`. For instance, `smotes/add-smtp-imposters`.
