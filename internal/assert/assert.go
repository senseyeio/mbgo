package assert

import (
	"reflect"
	"testing"
)

// Equals is a helper function used throughout the unit and integration
// tests to assert deep equality between an actual and expected value.
func Equals(tb testing.TB, actual, expected interface{}) {
	tb.Helper()

	if !reflect.DeepEqual(expected, actual) {
		tb.Fatalf("\n\n\texpected: %#v\n\n\tactual: %#v\n\n", expected, actual)
	}
}
