package kubernetes

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/api"
)

// ValidateKubernetesVersionUpgrade checks if a version is one of the allowed upgrade versions given current version
func ValidateKubernetesVersionUpgrade(newVersion string, currentVersion string) error {
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
