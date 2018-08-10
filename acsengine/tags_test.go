package acsengine

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/stretchr/testify/assert"
)

func TestValidateMaximumNumberOfARMTags(t *testing.T) {
	tagsMap := make(map[string]interface{})
	for i := 0; i < 16; i++ {
		tagsMap[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	_, es := validateAzureRMTags(tagsMap, "tags")

	assert.Equal(t, 1, len(es), "Expected one validation error for too many tags")

	assert.Contains(t, es[0].Error(), "a maximum of 15 tags", "Wrong validation error message for too many tags")
}

func TestValidateARMTagMaxKeyLength(t *testing.T) {
	tooLongKey := strings.Repeat("long", 128) + "a"
	tagsMap := make(map[string]interface{})
	tagsMap[tooLongKey] = "value"

	_, es := validateAzureRMTags(tagsMap, "tags")
	assert.Equal(t, 1, len(es), "Expected one validation error for a value which is > 512 chars")

	assert.Contains(t, es[0].Error(), "maximum length for a tag key", "Wrong validation error message maximum tag key length")

	assert.Contains(t, es[0].Error(), tooLongKey, "Expected validated error to contain the key name")

	assert.Contains(t, es[0].Error(), "513", "Expected the length in the validation error for tag key")
}

func TestValidateARMTagMaxValueLength(t *testing.T) {
	tagsMap := make(map[string]interface{})
	tagsMap["toolong"] = strings.Repeat("long", 64) + "a"

	_, es := validateAzureRMTags(tagsMap, "tags")
	assert.Equal(t, len(es), 1, "Expected one validation error for a value which is > 256 chars")

	assert.Contains(t, es[0].Error(), "maximum length for a tag value", "Wrong validation error message for maximum tag value length")

	assert.Contains(t, es[0].Error(), "toolong", "Expected validated error to contain the key name")

	assert.Contains(t, es[0].Error(), "257", "Expected the length in the validation error for value")
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

		assert.Equal(t, *expanded[k], strVal, "expanded value is incorrect")
	}
}

func TestExpandClusterTags(t *testing.T) {
	testData := make(map[string]interface{})
	testData["key1"] = "value1"
	testData["key2"] = 21
	testData["key3"] = "value3"

	expanded := expandClusterTags(testData)

	assert.Equal(t, len(expanded), 3, "Expected 3 results in expanded tag map, got %d", len(expanded))

	for k, v := range testData {
		var strVal string
		switch v.(type) {
		case string:
			strVal = v.(string)
		case int:
			strVal = fmt.Sprintf("%d", v.(int))
		}

		assert.Equal(t, expanded[k], strVal, "Expanded value %q incorrect: expected %q, got %q", k, strVal, expanded[k])
	}
}

func TestTagValueToString(t *testing.T) {
	vInt := 1
	value, err := tagValueToString(vInt)
	if err != nil {
		t.Fatalf("failed to convert value to string")
	}
	assert.Equal(t, value, "1")

	vStr := "hi"
	value, err = tagValueToString(vStr)
	if err != nil {
		t.Fatalf("failed to convert value to string")
	}
	assert.Equal(t, value, vStr)

	vMap := map[string]string{"good morning": "goodnight"}
	_, err = tagValueToString(vMap)
	if err == nil {
		t.Fatalf("should have failed to convert value to string")
	}
}

func TestFlattenTags(t *testing.T) {
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

	val, ok := output["Environment"]
	assert.True(t, ok, "output['Environment'] does not exist")
	assert.Equal(t, val, "Production", "output['Environment'] is not set correctly")
}

func TestFlattenTagsEmpty(t *testing.T) {
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

	assert.Equal(t, 0, len(output))
}

// func TestGetTags(t *testing.T) {
// 	r := resourceArmACSEngineKubernetesCluster()
// 	d := r.TestResourceData()

// 	tags := map[string]interface{}{}
// 	if err := d.Set("tags", tags); err != nil {
// 		t.Fatalf("set tags failed: %+v", err)
// 	}

// 	if err := getTags(d); err != nil {
// 		t.Fatalf("failed to get tags: %+v", err)
// 	}

// }

func TestSetTags(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

	tags := map[string]string{
		"home": "1111111111",
		"cell": "2222222222",
	}

	if err := setTags(d, tags); err != nil {
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
		resp, err := rgClient.Get(client.StopContext, resourceGroup)
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
