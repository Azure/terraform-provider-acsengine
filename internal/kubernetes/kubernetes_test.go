package kubernetes

import "testing"

func TestValidateKubernetesVersionUpgrade(t *testing.T) {
	cases := []struct {
		NewValue     string
		CurrentValue string
		ExpectError  bool
	}{
		{NewValue: "1.8.2", CurrentValue: "1.8.0", ExpectError: false},
		{NewValue: "1.8.2", CurrentValue: "1.8.2", ExpectError: true},
		{NewValue: "1.8.0", CurrentValue: "1.8.2", ExpectError: true},
		{NewValue: "1.11.0", CurrentValue: "1.8.2", ExpectError: true},
		{NewValue: "1.9.1", CurrentValue: "1.8.2", ExpectError: false},
		{NewValue: "1.10.0", CurrentValue: "1.8.0", ExpectError: true},
		{NewValue: "1.9.8", CurrentValue: "1.8.0", ExpectError: false},
		{NewValue: "1.8.1", CurrentValue: "1.7.12", ExpectError: false},
	}

	for _, tc := range cases {
		valid := true
		err := ValidateKubernetesVersionUpgrade(tc.NewValue, tc.CurrentValue)
		if err != nil {
			valid = false
		}
		if tc.ExpectError && valid {
			t.Fatalf("Expected the Kubernetes version validator to trigger an error for new version = '%s', current version = '%s'", tc.NewValue, tc.CurrentValue)
		} else if !tc.ExpectError && !valid {
			t.Fatalf("Expected the Kubernetes version validator to not trigger an error for new version = '%s', current version = '%s'. Instead got %+v", tc.NewValue, tc.CurrentValue, err)
		}
	}
}
