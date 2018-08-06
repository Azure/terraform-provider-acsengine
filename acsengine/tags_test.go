package acsengine

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestValidateMaximumNumberOfARMTags(t *testing.T) {
	tagsMap := make(map[string]interface{})
	for i := 0; i < 16; i++ {
		tagsMap[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	_, es := validateAzureRMTags(tagsMap, "tags")

	if len(es) != 1 {
		t.Fatal("Expected one validation error for too many tags")
	}

	if !strings.Contains(es[0].Error(), "a maximum of 15 tags") {
		t.Fatal("Wrong validation error message for too many tags")
	}
}

func TestValidateARMTagMaxKeyLength(t *testing.T) {
	tooLongKey := strings.Repeat("long", 128) + "a"
	tagsMap := make(map[string]interface{})
	tagsMap[tooLongKey] = "value"

	_, es := validateAzureRMTags(tagsMap, "tags")
	if len(es) != 1 {
		t.Fatal("Expected one validation error for a key which is > 512 chars")
	}

	if !strings.Contains(es[0].Error(), "maximum length for a tag key") {
		t.Fatal("Wrong validation error message maximum tag key length")
	}

	if !strings.Contains(es[0].Error(), tooLongKey) {
		t.Fatal("Expected validated error to contain the key name")
	}

	if !strings.Contains(es[0].Error(), "513") {
		t.Fatal("Expected the length in the validation error for tag key")
	}
}

func TestValidateARMTagMaxValueLength(t *testing.T) {
	tagsMap := make(map[string]interface{})
	tagsMap["toolong"] = strings.Repeat("long", 64) + "a"

	_, es := validateAzureRMTags(tagsMap, "tags")
	if len(es) != 1 {
		t.Fatal("Expected one validation error for a value which is > 256 chars")
	}

	if !strings.Contains(es[0].Error(), "maximum length for a tag value") {
		t.Fatal("Wrong validation error message for maximum tag value length")
	}

	if !strings.Contains(es[0].Error(), "toolong") {
		t.Fatal("Expected validated error to contain the key name")
	}

	if !strings.Contains(es[0].Error(), "257") {
		t.Fatal("Expected the length in the validation error for value")
	}
}

func TestExpandARMTags(t *testing.T) {
	testData := make(map[string]interface{})
	testData["key1"] = "value1"
	testData["key2"] = 21
	testData["key3"] = "value3"

	expanded := expandTags(testData)

	if len(expanded) != 3 {
		t.Fatalf("Expected 3 results in expanded tag map, got %d", len(expanded))
	}

	for k, v := range testData {
		var strVal string
		switch v.(type) {
		case string:
			strVal = v.(string)
		case int:
			strVal = fmt.Sprintf("%d", v.(int))
		}

		if *expanded[k] != strVal {
			t.Fatalf("Expanded value %q incorrect: expected %q, got %q", k, strVal, expanded[k])
		}
	}
}

func TestExpandClusterTags(t *testing.T) {
	testData := make(map[string]interface{})
	testData["key1"] = "value1"
	testData["key2"] = 21
	testData["key3"] = "value3"

	expanded := expandClusterTags(testData)

	if len(expanded) != 3 {
		t.Fatalf("Expected 3 results in expanded tag map, got %d", len(expanded))
	}

	for k, v := range testData {
		var strVal string
		switch v.(type) {
		case string:
			strVal = v.(string)
		case int:
			strVal = fmt.Sprintf("%d", v.(int))
		}

		if expanded[k] != strVal {
			t.Fatalf("Expanded value %q incorrect: expected %q, got %q", k, strVal, expanded[k])
		}
	}
}

func TestACSEngineK8sCluster_flattenTags(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenTags failed")
		}
	}()

	tags := map[string]string{
		"Environment": "Production",
	}

	output, err := flattenTags(tags)
	if err != nil {
		t.Fatalf("flattenTags failed: %v", err)
	}

	if _, ok := output["Environment"]; !ok {
		t.Fatalf("output['Environment'] does not exist")
	}
	if output["Environment"] != "Production" {
		t.Fatalf("output['Environment'] is not set correctly")
	}
}

func TestACSEngineK8sCluster_flattenTagsEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenTags failed")
		}
	}()

	tags := map[string]string{}

	output, err := flattenTags(tags)
	if err != nil {
		t.Fatalf("flattenTags failed: %v", err)
	}

	if len(output) != 0 {
		t.Fatalf("len(output) != 0")
	}
}

func TestACSEngineK8sCluster_setTags(t *testing.T) {
	r := resourceArmAcsEngineKubernetesCluster()
	d := r.TestResourceData()

	tags := map[string]string{
		"home": "1111111111",
		"cell": "2222222222",
	}

	err := setTags(d, tags)
	if err != nil {
		t.Fatalf("failed to set tags: %+v", err)
	}
}

func testCheckACSEngineClusterTagsExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		is, err := primaryInstanceState(s, name)
		if err != nil {
			return err
		}

		name := is.Attributes["name"]
		resourceGroup, hasResourceGroup := is.Attributes["resource_group"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Kubernetes cluster: %s", name)
		}

		client := testAccProvider.Meta().(*ArmClient)
		rgClient := client.resourceGroupsClient
		ctx := client.StopContext
		resp, err := rgClient.Get(ctx, resourceGroup)
		if err != nil {
			return fmt.Errorf("Error retrieving resource group: %+v", err)
		}
		if *resp.ID == "" {
			return fmt.Errorf("resource group ID is not set")
		}

		tag1, hasTag1 := is.Attributes["tags.Environment"]
		if !hasTag1 {
			return fmt.Errorf("Bad: no 'Environment' tag found in state for Kubernetes cluster: %s", name)
		}
		tag2, hasTag2 := is.Attributes["tags.Department"]
		if !hasTag2 {
			return fmt.Errorf("Bad: no 'Department' tag found in state for Kubernetes cluster: %s", name)
		}

		tagMap := resp.Tags
		if len(tagMap) != 2 {
			return fmt.Errorf("")
		}
		v, ok := tagMap["Environment"]
		if !ok {
			return fmt.Errorf("'Environment' tag not found not found in resource group %s", resourceGroup)
		}
		if *v != tag1 {
			return fmt.Errorf("'Environment' tag - actual: '%s', expected: '%s'", *v, tag1)
		}
		v, ok = tagMap["Department"]
		if !ok {
			return fmt.Errorf("'Department' tag not found in resource group %s", resourceGroup)
		}
		if *v != tag2 {
			return fmt.Errorf("'Department' tag - actual: '%s', expected: '%s'", *v, tag2)
		}

		return nil
	}
}
