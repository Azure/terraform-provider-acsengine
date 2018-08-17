package acsengine

import (
	"fmt"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/internal/resource"
	"github.com/stretchr/testify/assert"
)

func TestParseImportID(t *testing.T) {
	azureIDInput := "/subscriptions/1234/resourceGroups/testrg/providers/Microsoft.Resources/deployments/deploymentName"
	deploymentDirectoryInput := "_output/dnsPrefix"
	id := fmt.Sprintf("%s %s", azureIDInput, deploymentDirectoryInput)

	azureID, deploymentDirectory, err := parseImportID(id)
	if err != nil {
		t.Fatalf("parseImportID failed: %+v", err)
	}

	assert.Equal(t, azureID, azureIDInput, "parseImportID failed")
	if deploymentDirectory != deploymentDirectoryInput {
		t.Fatalf("parseImportID failed: deploymentDirectory was %s but expected %s", deploymentDirectory, deploymentDirectoryInput)
	}
	assert.Equal(t, deploymentDirectory, deploymentDirectoryInput, "parseImportID failed")

	if _, err = resource.ParseAzureResourceID(azureID); err != nil {
		t.Fatalf("failed to parse azureID: %+v", err)
	}
}

func TestParseInvalidImportID(t *testing.T) {
	cases := []struct {
		ImportID string
	}{
		{
			ImportID: "/subscriptions/1234/resourceGroups/testrg/providers/Microsoft.Resources/deployments/deploymentName",
		},
		{
			ImportID: "_output/dnsPrefix",
		},
	}

	for _, tc := range cases {
		_, _, err := parseImportID(tc.ImportID)
		if err == nil {
			t.Fatalf("parseImportID should have failed with ID %s", tc.ImportID)
		}
	}
}

func TestDeploymentNameAndResourceGroup(t *testing.T) {
	cases := []struct {
		azureID       string
		name          string
		resourceGroup string
	}{
		{
			azureID:       "/subscriptions/1234/resourceGroups/testrg/providers/Microsoft.Resources/deployments/deploymentName",
			name:          "deploymentName",
			resourceGroup: "testrg",
		},
	}

	for _, tc := range cases {
		name, resourceGroup, err := deploymentNameAndResourceGroup(tc.azureID)
		if err != nil {
			t.Fatalf("failed to get name and resource group from Azure ID: %+v", err)
		}

		assert.Equal(t, tc.name, name)
		assert.Equal(t, tc.resourceGroup, resourceGroup)
	}
}
