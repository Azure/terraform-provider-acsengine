package acsengine

import (
	"reflect"
	"testing"
)

func TestParseAzureResourceID(t *testing.T) {
	testCases := []struct {
		id                 string
		expectedResourceID *ResourceID
		expectError        bool
	}{
		{
			// Missing "resourceGroups".
			"/subscriptions/00000000-0000-0000-0000-000000000000//myResourceGroup/",
			nil,
			true,
		},
		{
			// Empty resource group ID.
			"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups//",
			nil,
			true,
		},
		{
			"random",
			nil,
			true,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
			nil,
			true,
		},
		{
			"subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
			nil,
			true,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1",
			&ResourceID{
				SubscriptionID: "6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
				ResourceGroup:  "testGroup1",
				Provider:       "",
				Path:           map[string]string{},
			},
			false,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1/providers/Microsoft.Network",
			&ResourceID{
				SubscriptionID: "6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
				ResourceGroup:  "testGroup1",
				Provider:       "Microsoft.Network",
				Path:           map[string]string{},
			},
			false,
		},
		{
			// Missing leading /
			"subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1/providers/Microsoft.Network/virtualNetworks/virtualNetwork1/",
			nil,
			true,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1/providers/Microsoft.Network/virtualNetworks/virtualNetwork1",
			&ResourceID{
				SubscriptionID: "6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
				ResourceGroup:  "testGroup1",
				Provider:       "Microsoft.Network",
				Path: map[string]string{
					"virtualNetworks": "virtualNetwork1",
				},
			},
			false,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1/providers/Microsoft.Network/virtualNetworks/virtualNetwork1?api-version=2006-01-02-preview",
			&ResourceID{
				SubscriptionID: "6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
				ResourceGroup:  "testGroup1",
				Provider:       "Microsoft.Network",
				Path: map[string]string{
					"virtualNetworks": "virtualNetwork1",
				},
			},
			false,
		},
		{
			"/subscriptions/6d74bdd2-9f84-11e5-9bd9-7831c1c4c038/resourceGroups/testGroup1/providers/Microsoft.Network/virtualNetworks/virtualNetwork1/subnets/publicInstances1?api-version=2006-01-02-preview",
			&ResourceID{
				SubscriptionID: "6d74bdd2-9f84-11e5-9bd9-7831c1c4c038",
				ResourceGroup:  "testGroup1",
				Provider:       "Microsoft.Network",
				Path: map[string]string{
					"virtualNetworks": "virtualNetwork1",
					"subnets":         "publicInstances1",
				},
			},
			false,
		},
		{
			"/subscriptions/34ca515c-4629-458e-bf7c-738d77e0d0ea/resourcegroups/acceptanceTestResourceGroup1/providers/Microsoft.Cdn/profiles/acceptanceTestCdnProfile1",
			&ResourceID{
				SubscriptionID: "34ca515c-4629-458e-bf7c-738d77e0d0ea",
				ResourceGroup:  "acceptanceTestResourceGroup1",
				Provider:       "Microsoft.Cdn",
				Path: map[string]string{
					"profiles": "acceptanceTestCdnProfile1",
				},
			},
			false,
		},
		{
			"/subscriptions/34ca515c-4629-458e-bf7c-738d77e0d0ea/resourceGroups/testGroup1/providers/Microsoft.ServiceBus/namespaces/testNamespace1/topics/testTopic1/subscriptions/testSubscription1",
			&ResourceID{
				SubscriptionID: "34ca515c-4629-458e-bf7c-738d77e0d0ea",
				ResourceGroup:  "testGroup1",
				Provider:       "Microsoft.ServiceBus",
				Path: map[string]string{
					"namespaces":    "testNamespace1",
					"topics":        "testTopic1",
					"subscriptions": "testSubscription1",
				},
			},
			false,
		},
	}

	for _, test := range testCases {
		parsed, err := parseAzureResourceID(test.id)
		if test.expectError && err != nil {
			continue
		}
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		if !reflect.DeepEqual(test.expectedResourceID, parsed) {
			t.Fatalf("Unexpected resource ID:\nExpected: %+v\nGot:      %+v\n", test.expectedResourceID, parsed)
		}
	}
}

func TestComposeAzureResourceID(t *testing.T) {
	testCases := []struct {
		resourceID  *ResourceID
		expectedID  string
		expectError bool
	}{
		{
			&ResourceID{
				SubscriptionID: "00000000-0000-0000-0000-000000000000",
				ResourceGroup:  "testGroup1",
				Provider:       "foo.bar",
				Path: map[string]string{
					"k1": "v1",
					"k2": "v2",
					"k3": "v3",
					"k4": "v4",
					"k5": "v5",
					"k6": "v6",
					"k7": "v7",
					"k8": "v8",
				},
			},
			"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/testGroup1/providers/foo.bar/k1/v1/k2/v2/k3/v3/k4/v4/k5/v5/k6/v6/k7/v7/k8/v8",
			false,
		},
		{
			&ResourceID{
				SubscriptionID: "00000000-0000-0000-0000-000000000000",
				ResourceGroup:  "testGroup1",
			},
			"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/testGroup1",
			false,
		},
		{
			// If Provider is specified, there must be at least one element in Path.
			&ResourceID{
				SubscriptionID: "00000000-0000-0000-0000-000000000000",
				ResourceGroup:  "testGroup1",
				Provider:       "foo.bar",
			},
			"",
			true,
		},
		{
			// One of the keys in Path is an empty string.
			&ResourceID{
				SubscriptionID: "00000000-0000-0000-0000-000000000000",
				ResourceGroup:  "testGroup1",
				Provider:       "foo.bar",
				Path: map[string]string{
					"k2": "v2",
					"":   "v1",
				},
			},
			"",
			true,
		},
		{
			// One of the values in Path is an empty string.
			&ResourceID{
				SubscriptionID: "00000000-0000-0000-0000-000000000000",
				ResourceGroup:  "testGroup1",
				Provider:       "foo.bar",
				Path: map[string]string{
					"k1": "v1",
					"k2": "",
				},
			},
			"",
			true,
		},
	}

	for _, test := range testCases {
		idString, err := composeAzureResourceID(test.resourceID)

		if test.expectError && err != nil {
			continue
		}

		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		if test.expectedID != idString {
			t.Fatalf("Unexpected resource ID string:\nExpected: %s\nGot:      %s\n", test.expectedID, idString)
		}
	}
}
