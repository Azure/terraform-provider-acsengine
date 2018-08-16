package acsengine

import (
	"fmt"
	"log"
	"path"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
)

func generateACSEngineTemplate(c *ArmClient, cluster Cluster, write bool) (string, string, error) {
	// I wonder if there's a more efficient way to generate certs than generating the entire templaet

	// certificates are generated here
	template, parameters, _ /*certsGenerated*/, err := cluster.formatTemplates(true)
	if err != nil {
		return "", "", fmt.Errorf("failed to format templates using cluster: %+v", err)
	}

	if write { // this should be default but allow for more testing
		if err = setCertificateProfileSecretsKeyVault(c, &cluster); err != nil {
			return "", "", fmt.Errorf("error setting keys and certificates in key vault: %+v", err)
		}

		if err = cluster.setCertificateProfileSecretsAPIModel(); err != nil {
			return "", "", fmt.Errorf("error setting cluster secret IDs: %+v", err)
		}

		var certsGenerated bool
		// generating templates again here so they contain key vault reference blocks instead of keys in plain text
		template, parameters, certsGenerated, err = cluster.formatTemplates(true)
		if err != nil {
			return "", "", fmt.Errorf("failed to format templates using cluster: %+v", err)
		}
		if certsGenerated {
			return "", "", fmt.Errorf("new certs should not have been generated")
		}

		deploymentDirectory := path.Join("_output", cluster.Properties.MasterProfile.DNSPrefix)
		if err = cluster.writeTemplatesAndCerts(template, parameters, deploymentDirectory, false); err != nil {
			return "", "", fmt.Errorf("error writing templates and certificates: %+v", err)
		}
	}

	return template, parameters, nil
}

func deployTemplate(c *ArmClient, name, resourceGroup, template, parameters string) (string, error) {
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

	if err := createDeployment(c, resourceGroup, name, &deployment); err != nil {
		return "", fmt.Errorf("failed to create deployment: %+v", err)
	}

	return getDeploymentID(c, resourceGroup, name)
}

func createDeployment(c *ArmClient, resourceGroup string, name string, deployment *resources.Deployment) error {
	deployClient := c.deploymentsClient
	future, err := deployClient.CreateOrUpdate(c.StopContext, resourceGroup, name, *deployment)
	if err != nil {
		return fmt.Errorf("error creating deployment: %+v", err)
	}
	log.Println("[INFO] Deployment created (1)") // log

	if err = future.WaitForCompletion(c.StopContext, deployClient.Client); err != nil {
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

func getDeploymentID(c *ArmClient, resourceGroup string, name string) (string, error) {
	deployClient := c.deploymentsClient
	read, err := deployClient.Get(c.StopContext, resourceGroup, name)
	if err != nil {
		return "", fmt.Errorf("error getting deployment: %+v", err)
	}
	if read.ID == nil {
		return "", fmt.Errorf("Cannot read ACS Engine Kubernetes cluster deployment %s (resource group %s) ID", name, resourceGroup)
	}
	log.Printf("[INFO] cluster %q ID: %q\n", name, *read.ID)

	return *read.ID, nil
}
