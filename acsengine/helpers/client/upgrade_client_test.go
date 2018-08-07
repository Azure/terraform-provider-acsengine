package client

import "testing"

func TestUpgradeValidate(t *testing.T) {
	cases := []struct {
		Client      UpgradeClient
		ExpectError bool
	}{
		{
			Client:      UpgradeClient{},
			ExpectError: true,
		},
		{
			Client: UpgradeClient{
				ACSEngineClient: ACSEngineClient{
					ResourceGroupName:   "rg",
					Location:            "westus",
					DeploymentDirectory: "directory",
				},
			},
			ExpectError: true,
		},
		{
			Client: UpgradeClient{
				ACSEngineClient: ACSEngineClient{
					ResourceGroupName:   "rg",
					Location:            "westus",
					DeploymentDirectory: "directory",
				},
				UpgradeVersion: "1.8.13",
			},
			ExpectError: false,
		},
	}

	for _, tc := range cases {
		err := tc.Client.Validate()
		if err == nil && tc.ExpectError {
			t.Fatalf("expected error")
		}
		if err != nil && !tc.ExpectError {
			t.Fatalf("error: %+v", err)
		}
	}
}
