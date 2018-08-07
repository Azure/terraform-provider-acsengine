package test

import (
	"reflect"
	"strings"
	"testing"
)

// Equals tests whether actual == expected
func Equals(t *testing.T, actual interface{}, expected interface{}) {
	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Fatalf("")
	}
	switch act := actual.(type) {
	case int:
		intEquals(t, act, expected.(int))
	case string:
		stringEquals(t, act, expected.(string))
	default:
		t.Fatalf("unrecognized type")
	}
}

// Contains tests whether a string contains a substring
func Contains(t *testing.T, str string, sub string) {
	if !strings.Contains(str, sub) {
		t.Fatalf("string '%s' does not contain '%s'", str, sub)
	}
}

// OK fails if not okay with error string
func OK(t *testing.T, ok bool, message string) {
	if !ok {
		t.Fatalf(message)
	}
}

func intEquals(t *testing.T, actual int, expected int) {
	if actual != expected {
		t.Fatalf("equals check failed - actual: '%d', expected: '%d'", actual, expected)
	}
}

func stringEquals(t *testing.T, actual string, expected string) {
	if actual != expected {
		t.Fatalf("equals check failed - actual: '%s', expected: '%s'", actual, expected)
	}
}
