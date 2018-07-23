package acsengine

import (
	"strconv"
	"testing"
)

func TestValidateIntInSlice(t *testing.T) {

	cases := []struct {
		Input  []int
		Value  int
		Errors int
	}{
		{
			Input:  []int{},
			Value:  0,
			Errors: 1,
		},
		{
			Input:  []int{1},
			Value:  1,
			Errors: 0,
		},
		{
			Input:  []int{1, 2, 3, 4, 5},
			Value:  3,
			Errors: 0,
		},
		{
			Input:  []int{1, 3, 5},
			Value:  3,
			Errors: 0,
		},
		{
			Input:  []int{1, 3, 5},
			Value:  4,
			Errors: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateIntInSlice(tc.Input)(tc.Value, "azurerm_postgresql_database")

		if len(errors) != tc.Errors {
			t.Fatalf("Expected the validateIntInSlice trigger a validation error for input: %+v looking for %+v", tc.Input, tc.Value)
		}
	}

}

func TestValidateIntBetweenDivisibleBy(t *testing.T) {
	cases := []struct {
		Min    int
		Max    int
		Div    int
		Value  interface{}
		Errors int
	}{
		{
			Min:    1025,
			Max:    2048,
			Div:    1024,
			Value:  1024,
			Errors: 1,
		},
		{
			Min:    1025,
			Max:    2048,
			Div:    3,
			Value:  1024,
			Errors: 1,
		},
		{
			Min:    1024,
			Max:    2048,
			Div:    1024,
			Value:  3072,
			Errors: 1,
		},
		{
			Min:    1024,
			Max:    2048,
			Div:    1024,
			Value:  2049,
			Errors: 1,
		},
		{
			Min:    1024,
			Max:    2048,
			Div:    1024,
			Value:  1024,
			Errors: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateIntBetweenDivisibleBy(tc.Min, tc.Max, tc.Div)(tc.Value, strconv.Itoa(tc.Value.(int)))
		if len(errors) != tc.Errors {
			t.Fatalf("Expected intBetweenDivisibleBy to trigger '%d' errors for '%s' - got '%d'", tc.Errors, tc.Value, len(errors))
		}
	}
}

func TestValidateCollation(t *testing.T) {
	cases := []struct {
		Value  string
		Errors int
	}{
		{
			Value:  "en-US",
			Errors: 1,
		},
		{
			Value:  "en_US",
			Errors: 0,
		},
		{
			Value:  "en US",
			Errors: 0,
		},
		{
			Value:  "English_United States.1252",
			Errors: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateCollation()(tc.Value, "collation")
		if len(errors) != tc.Errors {
			t.Fatalf("Expected validateCollation to trigger '%d' errors for '%s' - got '%d'", tc.Errors, tc.Value, len(errors))
		}
	}
}
