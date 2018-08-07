package client

import "testing"

func TestScaleValidate(t *testing.T) {
	cases := []struct {
		Client      ScaleClient
		ExpectError bool
	}{
		{
			Client:      ScaleClient{},
			ExpectError: true,
		},
		{
			Client: ScaleClient{
				ACSEngineClient: ACSEngineClient{
					ResourceGroupName:   "rg",
					Location:            "westus",
					DeploymentDirectory: "directory",
				},
			},
			ExpectError: true,
		},
		{
			Client: ScaleClient{
				ACSEngineClient: ACSEngineClient{
					ResourceGroupName:   "rg",
					Location:            "westus",
					DeploymentDirectory: "directory",
				},
				DesiredAgentCount: 1,
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
