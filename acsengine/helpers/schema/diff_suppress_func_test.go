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
