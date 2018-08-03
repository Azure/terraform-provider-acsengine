package acsengine

import "testing"

func TestAzureRMNormalizeLocation(t *testing.T) {
	s := azureRMNormalizeLocation("West US")
	if s != "westus" {
		t.Fatalf("expected location to equal westus, actual %s", s)
	}
}

func TestAzureRMSuppressLocationDiff(t *testing.T) {
	cases := []struct {
		New      string
		Old      string
		Expected bool
	}{
		{
			New:      "West US",
			Old:      "westus2",
			Expected: false,
		},
		{
			New:      "South Central US",
			Old:      "southcentralus",
			Expected: true,
		},
	}

	for _, tc := range cases {
		diff := azureRMSuppressLocationDiff("", tc.Old, tc.New, nil)

		if diff != tc.Expected {
			t.Errorf("%s == %s - actual: %t, expected: %t", tc.Old, tc.New, diff, tc.Expected)
		}
	}
}
