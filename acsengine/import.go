package acsengine

import (
	"fmt"
	"log"
	"strings"

	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/helper/schema"
)

// the ID passed will be a string of format "AZURE_RESOURCE_ID*space*APIMODEL_DIRECTORY"
func resourceACSEngineK8sClusterImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client := m.(*ArmClient)
	deployClient := client.deploymentsClient

	azureID, deploymentDirectory, err := parseImportID(d.Id())
	if err != nil {
		return nil, err
	}

	id, err := utils.ParseAzureResourceID(azureID)
	if err != nil {
		return nil, err
	}
	name := id.Path["deployments"]
	if name == "" {
		name = id.Path["Deployments"]
	}
	resourceGroup := id.ResourceGroup

	read, err := deployClient.Get(client.StopContext, resourceGroup, name)
	if err != nil {
		return nil, fmt.Errorf("error getting deployment: %+v", err)
	}
	if read.ID == nil {
		return nil, fmt.Errorf("Cannot read ACS Engine Kubernetes cluster deployment %s (resource group %s) ID", name, resourceGroup)
	}
	log.Printf("[INFO] cluster %q ID: %q", name, *read.ID)

	d.SetId(*read.ID)

	apimodel, err := getAPIModelFromFile(deploymentDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to get apimodel.json: %+v", err)
	}
	if err := d.Set("api_model", apimodel); err != nil {
		return nil, fmt.Errorf("failed to set `api_model`: %+v", err)
	}

	return []*schema.ResourceData{d}, nil
}

func parseImportID(dID string) (string, string, error) {
	input := strings.Split(dID, " ")
	if len(input) != 2 {
		return "", "", fmt.Errorf("split import ID is wrong length: expected 2 but got %d", len(input))
	}

	azureID := input[0]
	deploymentDirectory := input[1]

	return azureID, deploymentDirectory, nil
}
