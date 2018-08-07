package acsengine

import (
	"fmt"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
)

func TestACSEngineK8sCluster_parseImportID(t *testing.T) {
	azureIDInput := "/subscriptions/1234/resourceGroups/testrg/providers/Microsoft.Resources/deployments/deploymentName"
	deploymentDirectoryInput := "_output/dnsPrefix"
	id := fmt.Sprintf("%s %s", azureIDInput, deploymentDirectoryInput)

	azureID, deploymentDirectory, err := parseImportID(id)
	if err != nil {
		t.Fatalf("parseImportID failed: %+v", err)
	}

	if azureID != azureIDInput {
		t.Fatalf("parseImportID failed: azureID was %s but expected %s", azureID, azureIDInput)
	}
	if deploymentDirectory != deploymentDirectoryInput {
		t.Fatalf("parseImportID failed: deploymentDirectory was %s but expected %s", deploymentDirectory, deploymentDirectoryInput)
	}

	if _, err = utils.ParseAzureResourceID(azureID); err != nil {
		t.Fatalf("failed to parse azureID: %+v", err)
	}
}
