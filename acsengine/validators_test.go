package acsengine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

		assert.Equal(t, len(errors), tc.ErrCount, fmt.Sprintf("Expected the Azure RM Kubernetes cluster master profile count to trigger a validation error for '%d'", tc.Value))
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

		assert.Equal(t, len(errors), tc.ErrCount, fmt.Sprintf("Expected the Azure RM Kubernetes cluster agent pool profile Count to trigger a validation error for '%d'", tc.Value))
	}
}
