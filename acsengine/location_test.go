package acsengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzureRMNormalizeLocation(t *testing.T) {
	s := azureRMNormalizeLocation("West US")
	assert.Equal(t, s, "westus", "location not normalized correctly")
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

		assert.Equal(t, diff, tc.Expected, "%s == %s", tc.Old, tc.New)
	}
}
