package testutil

import (
	"reflect"
	"testing"
)

// ExpectEqual is a helper function used throughout the unit and integration
// tests to assert deep equality between an actual and expected value.
func ExpectEqual(t *testing.T, actual, expected interface{}) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("\nexpected:\n\t%+v\nto equal:\n\t%+v\n", actual, expected)
	}
}
