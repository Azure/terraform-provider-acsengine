package acsengine

import "testing"

func TestACSEngineK8sCluster_validateKubernetesVersion(t *testing.T) {
	cases := []struct {
		Version     string
		ExpectError bool
	}{
		{Version: "1.8.2", ExpectError: false},
		{Version: "3.0.0", ExpectError: true},
		{Version: "1.7.12", ExpectError: false},
		{Version: "181", ExpectError: true},
		{Version: "2.18.3", ExpectError: true},
	}

	for _, tc := range cases {
		_, errors := validateKubernetesVersion(tc.Version, "")
		if !tc.ExpectError && len(errors) > 0 {
			t.Fatalf("Version %s should not have failed", tc.Version)
		}
		if tc.ExpectError && len(errors) == 0 {
			t.Fatalf("Version %s should have failed", tc.Version)
		}
	}
}

func TestACSEngineK8sCluster_validateKubernetesVersionUpgrade(t *testing.T) {
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
		err := validateKubernetesVersionUpgrade(tc.NewValue, tc.CurrentValue)
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

// correct values are 1, 3, and 5
func TestAccACSEngineK8sCluster_masterProfileCountValidation(t *testing.T) {
	cases := []struct {
		Value    int
		ErrCount int
	}{
		{Value: 0, ErrCount: 1},
		{Value: 1, ErrCount: 0},
		{Value: 2, ErrCount: 1},
		{Value: 3, ErrCount: 0},
		{Value: 4, ErrCount: 1},
		{Value: 5, ErrCount: 0},
		{Value: 6, ErrCount: 1},
	}

	for _, tc := range cases { // for each test case
		// from resource_arm_container_service.go
		_, errors := validateMasterProfileCount(tc.Value, "acsengine_kubernetes_cluster")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Kubernetes cluster master profile count to trigger a validation error for '%d'", tc.Value)
		}
	}
}

// correct values are 1-100, can be even or odd
func TestAccACSEngineK8sCluster_agentPoolProfileCountValidation(t *testing.T) {
	cases := []struct {
		Value    int
		ErrCount int
	}{
		{Value: 0, ErrCount: 1},
		{Value: 1, ErrCount: 0},
		{Value: 2, ErrCount: 0},
		{Value: 99, ErrCount: 0},
		{Value: 100, ErrCount: 0},
		{Value: 101, ErrCount: 1},
	}

	for _, tc := range cases { // for each test case
		// from resource_arm_container_service.go
		_, errors := validateAgentPoolProfileCount(tc.Value, "acsengine_kubernetes_cluster")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Kubernetes cluster agent pool profile Count to trigger a validation error for '%d'", tc.Value)
		}
	}
}
