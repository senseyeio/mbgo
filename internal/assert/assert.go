// Copyright (c) 2018 Senseye Ltd. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

// Package assert is used internally by tests to make basic assertions.
package assert

import (
	"reflect"
	"testing"
)

// Equals is a helper function used throughout the unit and integration
// tests to assert deep equality between an actual and expected value.
func Equals(tb testing.TB, expected, actual interface{}) {
	tb.Helper()

	if !reflect.DeepEqual(expected, actual) {
		tb.Errorf("\n\n\texpected: %#v\n\n\tactual: %#v\n\n", expected, actual)
	}
}

// Ok fails the test if an err is not nil.
func Ok(tb testing.TB, err error) {
	tb.Helper()

	if err != nil {
		tb.Errorf("unexpected error: %#v\n\n", err)
	}
}

// MustOk fails the test now if an err is not nil.
func MustOk(tb testing.TB, err error) {
	tb.Helper()

	if err != nil {
		tb.Fatalf("fatal error: %#v\n\n", err)
	}
}
