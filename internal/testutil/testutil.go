package testutil

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// ExpectEqual is a helper function used throughout the unit and integration
// tests to assert deep equality between an actual and expected value.
func ExpectEqual(tb testing.TB, actual, expected interface{}) {
	if !reflect.DeepEqual(actual, expected) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, expected, actual)
		tb.FailNow()
	}
}
