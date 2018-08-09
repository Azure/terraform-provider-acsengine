package acsengine

import (
	"fmt"
	"path"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/hashicorp/terraform/helper/schema"
)

func generateACSEngineTemplate(d *schema.ResourceData, write bool) (template string, parameters string, err error) {
	cluster, err := initializeContainerService(d)
	if err != nil {
		return "", "", err
	}

	template, parameters, certsGenerated, err := formatTemplates(cluster, true)
	if err != nil {
		return "", "", fmt.Errorf("failed to format templates using cluster: %+v", err)
	}

	if write { // this should be default but allow for more testing
		deploymentDirectory := path.Join("_output", cluster.Properties.MasterProfile.DNSPrefix)
		if err = writeTemplatesAndCerts(cluster, template, parameters, deploymentDirectory, certsGenerated); err != nil {
			return "", "", fmt.Errorf("error writing templates and certificates: %+v", err)
		}
	}
	if err = setAPIModel(d, cluster); err != nil {
		return "", "", fmt.Errorf("error setting API model: %+v", err)
	}

	return template, parameters, nil
}

func deployTemplate(d *schema.ResourceData, client *ArmClient, template, parameters string) (id string, err error) {
	deployClient := client.deploymentsClient
	ctx := client.StopContext

	var name, resourceGroup string
	var v interface{}
	var ok bool

	if v, ok = d.GetOk("name"); !ok {
		return "", fmt.Errorf("cluster 'name' not found")
	}
	name = v.(string)

	if v, ok = d.GetOk("resource_group"); !ok {
		return "", fmt.Errorf("cluster 'resource_group' not found")
	}
	resourceGroup = v.(string)

	azureDeployTemplate, azureDeployParameters, err := expandTemplates(template, parameters)
	if err != nil {
		return "", fmt.Errorf("failed to expand template and parameters: %+v", err)
	}

	properties := resources.DeploymentProperties{
		Mode:       resources.Incremental,
		Parameters: azureDeployParameters["parameters"],
		Template:   azureDeployTemplate,
	}

	deployment := resources.Deployment{
		Properties: &properties,
	}

	future, err := deployClient.CreateOrUpdate(ctx, resourceGroup, name, deployment)
	if err != nil {
		return "", fmt.Errorf("error creating deployment: %+v", err)
	}
	fmt.Println("Deployment created (1)")

	if err = future.WaitForCompletion(ctx, deployClient.Client); err != nil {
		return "", fmt.Errorf("error creating deployment: %+v", err)
	}

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
