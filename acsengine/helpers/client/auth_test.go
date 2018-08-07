package client

import "testing"

func TestValidateAuthArgs(t *testing.T) {
	cases := []struct {
		Auth        AuthArgs
		ExpectError bool
	}{
		{
			Auth:        AuthArgs{},
			ExpectError: true,
		},
		// {
		// 	Auth: AuthArgs{
		// 		RawSubscriptionID: "12345678-9000-1000-1100-120000000000",
		// 		RawClientID:       "12345678-9000-1000-1100-120000000000",
		// 	},
		// 	ExpectError: false,
		// },
		{
			Auth: AuthArgs{
				RawSubscriptionID: "12345678-9000-1000-1100-120000000000",
				AuthMethod:        "client_secret",
				RawClientID:       "12345678-9000-1000-1100-120000000000",
			},
			ExpectError: true,
		},
		// {
		// 	Auth: AuthArgs{
		// 		RawSubscriptionID: "12345678-9000-1000-1100-120000000000",
		// 		AuthMethod:        "client_secret",
		// 		RawClientID:       "1234",
		// 		ClientSecret:      "12345678-9000-1000-1100-120000000000",
		// 	},
		// 	ExpectError: false,
		// },
	}
	for _, tc := range cases {
		AddAuthArgs(&tc.Auth)
		err := tc.Auth.ValidateAuthArgs()
		if err == nil && tc.ExpectError {
			t.Fatalf("expected error")
		}
		if err != nil && !tc.ExpectError {
			t.Fatalf("error: %+v", err)
		}
	}
}
