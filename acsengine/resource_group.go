package acsengine

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGroupNameSchema() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		DiffSuppressFunc: resourceAzurermResourceGroupNameDiffSuppress,
		ValidateFunc:     validateArmResourceGroupName,
	}
}

func resourceGroupNameForDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	}
}

func validateArmResourceGroupName(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)

	if len(value) > 80 {
		es = append(es, fmt.Errorf("%q may not exceed 80 characters in length", k))
	}

	if strings.HasSuffix(value, ".") {
		es = append(es, fmt.Errorf("%q may not end with a period", k))
	}

	// regex pulled from https://docs.microsoft.com/en-us/rest/api/resources/resourcegroups/createorupdate
	if matched := regexp.MustCompile(`^[-\w\._\(\)]+$`).Match([]byte(value)); !matched {
		es = append(es, fmt.Errorf("%q may only contain alphanumeric characters, dash, underscores, parentheses and periods", k))
	}

	return
}

// Resource group names can be capitalised, but we store them in lowercase.
// Use a custom diff function to avoid creation of new resources.
func resourceAzurermResourceGroupNameDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	return strings.ToLower(old) == strings.ToLower(new)
}

// is this the right place for this function?

// Creates a resource group which will be used for all of the cluster resources
func createClusterResourceGroup(d *schema.ResourceData, m interface{}) error {
	client := m.(*ArmClient)
	rgClient := client.resourceGroupsClient
	ctx := client.StopContext

	var v interface{}
	var ok bool
	var name, location string

	v, ok = d.GetOk("resource_group")
	if !ok {
		return fmt.Errorf("cluster 'resource_group' not found")
	}
	name = v.(string)

	v, ok = d.GetOk("location")
	if !ok {
		return fmt.Errorf("cluster 'location' not found")
	}
	location = azureRMNormalizeLocation(v.(string))

	var tags map[string]interface{}
	if v, ok := d.GetOk("tags"); ok {
		tags = v.(map[string]interface{})
	} else {
		tags = map[string]interface{}{}
	}
	parameters := resources.Group{
		Location: utils.String(location),
		Tags:     expandTags(tags),
	}
	_, err := rgClient.CreateOrUpdate(ctx, name, parameters)
	if err != nil {
		return fmt.Errorf("Error creating resource group: %+v", err)
	}

	resp, err := rgClient.Get(ctx, name)
	if err != nil {
		return fmt.Errorf("Error retrieving resource group: %+v", err)
	}
	if *resp.ID == "" {
		return fmt.Errorf("resource group ID is not set")
	}
	log.Printf("[INFO] resource group %q ID: %q", name, *resp.ID)

	return nil
}
