package acsengine

import (
	"context"
	"fmt"
	"log"
	"path"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
)

func generateACSEngineTemplate(cluster Cluster, write bool) (template string, parameters string, err error) {
	template, parameters, certsGenerated, err := cluster.formatTemplates(true)
	if err != nil {
		return "", "", fmt.Errorf("failed to format templates using cluster: %+v", err)
	}

	// now replace lines in templates with certificate/key IDs?
	// I don't think I can generate a template with keys in it since they don't exist yet

	if write { // this should be default but allow for more testing
		deploymentDirectory := path.Join("_output", cluster.Properties.MasterProfile.DNSPrefix)
		if err = cluster.writeTemplatesAndCerts(template, parameters, deploymentDirectory, certsGenerated); err != nil {
			return "", "", fmt.Errorf("error writing templates and certificates: %+v", err)
		}
	}

	return template, parameters, nil
}

func deployTemplate(client *ArmClient, name, resourceGroup, template, parameters string) (id string, err error) {
	azureDeployTemplate, azureDeployParametersFile, err := expandTemplates(template, parameters)
	if err != nil {
		return "", fmt.Errorf("failed to expand template and parameters: %+v", err)
	}

	v, ok := azureDeployParametersFile["parameters"]
	if !ok {
		return "", fmt.Errorf("azureDeployParameters formatted incorrectly")
	}
	azureDeployParameters := v.(map[string]interface{})

	deployment := resources.Deployment{
		Properties: &resources.DeploymentProperties{
			Mode:       resources.Incremental,
			Parameters: azureDeployParameters,
			Template:   azureDeployTemplate,
		},
	}

	if err := createDeployment(client.StopContext, client, resourceGroup, name, &deployment); err != nil {
		return "", fmt.Errorf("failed to create deployment: %+v", err)
	}

	return getDeploymentID(client.StopContext, client, resourceGroup, name)
}

func createDeployment(ctx context.Context, client *ArmClient, resourceGroup string, name string, deployment *resources.Deployment) error {
	deployClient := client.deploymentsClient
	future, err := deployClient.CreateOrUpdate(ctx, resourceGroup, name, *deployment)
	if err != nil {
		return fmt.Errorf("error creating deployment: %+v", err)
	}
	fmt.Println("[INFO] Deployment created (1)") // log

	if err = future.WaitForCompletion(client.StopContext, deployClient.Client); err != nil {
		return fmt.Errorf("error creating deployment: %+v", err)
	}
	_, err = future.Result(deployClient)
	if err != nil {
		return fmt.Errorf("error getting deployment result")
	}
	// check response status code
	log.Println("[INFO] Deployment successful")

	return nil
}

func getDeploymentID(ctx context.Context, client *ArmClient, resourceGroup string, name string) (string, error) {
	deployClient := client.deploymentsClient
	read, err := deployClient.Get(ctx, resourceGroup, name)
	if err != nil {
		return "", fmt.Errorf("error getting deployment: %+v", err)
	}
	if read.ID == nil {
		return "", fmt.Errorf("Cannot read ACS Engine Kubernetes cluster deployment %s (resource group %s) ID", name, resourceGroup)
	}
	fmt.Printf("[INFO] cluster %q ID: %q\n", name, *read.ID)

	return *read.ID, nil
}
