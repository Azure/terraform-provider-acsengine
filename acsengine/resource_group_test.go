package acsengine

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
)

func TestValidateArmResourceGroupName(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "",
			ErrCount: 1,
		},
		{
			Value:    "hello",
			ErrCount: 0,
		},
		{
			Value:    "Hello",
			ErrCount: 0,
		},
		{
			Value:    "hello-world",
			ErrCount: 0,
		},
		{
			Value:    "Hello_World",
			ErrCount: 0,
		},
		{
			Value:    "HelloWithNumbers12345",
			ErrCount: 0,
		},
		{
			Value:    "(Did)You(Know)That(Brackets)Are(Allowed)In(Resource)Group(Names)",
			ErrCount: 0,
		},
		{
			Value:    "EndingWithAPeriod.",
			ErrCount: 1,
		},
		{
			Value:    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/foo",
			ErrCount: 1,
		},
		{
			Value:    acctest.RandString(80),
			ErrCount: 0,
		},
		{
			Value:    acctest.RandString(81),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateArmResourceGroupName(tc.Value, "azurerm_resource_group")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected validateArmResourceGroupName to trigger '%d' errors for '%s' - got '%d'", tc.ErrCount, tc.Value, len(errors))
		}
	}
}

func TestResourceGroupNameDiffSuppress(t *testing.T) {
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
		diff := resourceAzurermResourceGroupNameDiffSuppress("", tc.Old, tc.New, nil)

		if diff != tc.Expected {
			t.Fatalf("")
		}
	}

}
