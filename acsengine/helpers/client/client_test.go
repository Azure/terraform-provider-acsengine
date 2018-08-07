package client

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		Client      ACSEngineClient
		ExpectError bool
	}{
		{
			Client:      ACSEngineClient{},
			ExpectError: true,
		},
		{
			Client: ACSEngineClient{
				ResourceGroupName: "rg",
			},
			ExpectError: true,
		},
		{
			Client: ACSEngineClient{
				ResourceGroupName: "rg",
				Location:          "westus",
			},
			ExpectError: true,
		},
		{
			Client: ACSEngineClient{
				ResourceGroupName:   "rg",
				Location:            "westus",
				DeploymentDirectory: "directory",
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
