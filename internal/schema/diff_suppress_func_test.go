package schema

import "testing"

func TestIgnoreCaseDiffSuppressFunc(t *testing.T) {
	cases := []struct {
		New      string
		Old      string
		Expected bool
	}{
		{
			New:      "testRG",
			Old:      "testrg",
			Expected: true,
		},
		{
			New:      "testrg1",
			Old:      "testrg",
			Expected: false,
		},
	}

	for _, tc := range cases {
		diff := IgnoreCaseDiffSuppressFunc("", tc.Old, tc.New, nil)

		if diff != tc.Expected {
			t.Fatalf("")
		}
	}

}

func TestIgnoreCaseStateFunc(t *testing.T) {
	cases := []struct {
		Value    string
		Expected string
	}{
		{
			Value:    "VaLuE",
			Expected: "value",
		},
		{
			Value:    "testrg",
			Expected: "testrg",
		},
	}

	for _, tc := range cases {
		output := IgnoreCaseStateFunc(tc.Value)

		if output != tc.Expected {
			t.Fatalf("")
		}
	}
}
