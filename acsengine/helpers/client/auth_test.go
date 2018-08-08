package client

import "testing"

func TestValidateAuthArgs(t *testing.T) {
	cases := []struct {
		RawSubscriptionID string
		RawClientID       string
		ClientSecret      string
		AuthMethod        string
		ExpectError       bool
	}{
		{
			RawSubscriptionID: "",
			RawClientID:       "",
			ExpectError:       true,
		},
		{
			RawSubscriptionID: "12345678-9000-1000-1100-120000000000",
			RawClientID:       "12345678-9000-1000-1100-120000000000",
			ExpectError:       false,
		},
		{
			RawSubscriptionID: "12345678-9000-1000-1100-120000000000",
			RawClientID:       "12345678-9000-1000-1100-120000000000",
			AuthMethod:        "client_secret",
			ExpectError:       true,
		},
		{
			RawSubscriptionID: "12345678-9000-1000-1100-120000000000",
			AuthMethod:        "client_secret",
			RawClientID:       "12345678-9000-1000-1100-120000000000",
			ClientSecret:      "12345678-9000-1000-1100-120000000000",
			ExpectError:       false,
		},
	}

	for _, tc := range cases {
		auth := AuthArgs{}
		AddAuthArgs(&auth)
		auth.RawSubscriptionID = tc.RawSubscriptionID
		auth.RawClientID = tc.RawClientID
		auth.ClientSecret = tc.ClientSecret
		auth.AuthMethod = tc.AuthMethod
		err := auth.ValidateAuthArgs()
		if err == nil && tc.ExpectError {
			t.Fatalf("expected error")
		}
		if err != nil && !tc.ExpectError {
			t.Fatalf("error: %+v", err)
		}
	}
}
