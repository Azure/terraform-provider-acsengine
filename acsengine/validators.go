package acsengine

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/common"
)

func validateKubernetesVersion(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	capacities := common.AllKubernetesSupportedVersions

	if !capacities[value] {
		errors = append(errors, fmt.Errorf("ACS Engine Kubernetes Cluster: Kubernetes version %s is invalid or not supported", value))
	}
	return
}

func validateKubernetesVersionUpgrade(newVersion string, currentVersion string) error {
	kubernetesProfile := api.OrchestratorProfile{
		OrchestratorType:    "Kubernetes",
		OrchestratorVersion: currentVersion,
	}
	kubernetesInfo, err := api.GetOrchestratorVersionProfile(&kubernetesProfile)
	if err != nil {
		return fmt.Errorf("error getting a list of the available upgrades: %+v", err)
	}
	found := false
	for _, up := range kubernetesInfo.Upgrades { // checking that version I want is within the allowed versions
		if up.OrchestratorVersion == newVersion {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("version %s is not supported (either doesn't exist, is a downgrade or same version, or is an upgrade by more than 1 minor version)", newVersion)
	}

	return nil
}

func validateMasterProfileCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	capacities := map[int]bool{
		1: true,
		3: true,
		5: true,
	}

	if !capacities[value] {
		errors = append(errors, fmt.Errorf("the number of master nodes must be 1, 3 or 5"))
	}
	return
}

func validateAgentPoolProfileCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value > 100 || value <= 0 {
		errors = append(errors, fmt.Errorf("the count for an agent pool profile can only be between 1 and 100"))
	}
	return
}
