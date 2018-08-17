package acsengine

import (
	"fmt"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/internal/tester"
	"github.com/stretchr/testify/assert"
)

func TestAddValue(t *testing.T) {
	parameters := map[string]interface{}{}

	addValue(parameters, "key", "data")

	v, ok := parameters["key"]
	assert.True(t, ok, "could not find key")
	val := v.(map[string]interface{})
	assert.Equal(t, val["value"], "data", "value not set correctly")
}

func TestExpandTemplateBodies(t *testing.T) {
	body := `{
		"groceries": {
			"quinoa": "5",
			"pasta": "2"
		}
	}`

	template, parameters, err := expandTemplates(body, body)
	if err != nil {
		t.Fatalf("expand templates failed: %+v", err)
	}

	v, ok := parameters["groceries"]
	assert.True(t, ok, "could not find `groceries`")
	paramsGroceries := v.(map[string]interface{})
	assert.Equal(t, len(paramsGroceries), 2, fmt.Sprintf("length of grocery list is not correct: expected 2 and found %d", len(paramsGroceries)))
	v, ok = paramsGroceries["quinoa"]
	assert.True(t, ok, "could not find `quinoa`")
	assert.Equal(t, v.(string), "5")

	v, ok = template["groceries"]
	assert.True(t, ok, "could not find `groceries`")
	templateGroceries := v.(map[string]interface{})
	if len(templateGroceries) != 2 {
		t.Fatalf("length of grocery list is not correct: expected 2 and found %d", len(templateGroceries))
	}
	assert.Equal(t, len(templateGroceries), 2)
	v, ok = templateGroceries["pasta"]
	assert.True(t, ok, "could not find `pasta`")
	assert.Equal(t, v.(string), "2")
}

func TestExpandBody(t *testing.T) {
	body := `{
		"groceries": {
			"bananas": "5",
			"pasta": "2"
		}
	}`

	expandedBody, err := expandBody(body)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	v, ok := expandedBody["groceries"]
	assert.True(t, ok, "could not find `groceries`")
	groceries := v.(map[string]interface{})
	assert.Equal(t, len(groceries), 2)
	v, ok = groceries["bananas"]
	assert.True(t, ok, "could not find `bananas`")
	assert.Equal(t, v.(string), "5")
}

func TestExpandBodyBad(t *testing.T) {
	body := `{
		"groceries": {
			"bananas": "5",
			"pasta": "2",
		},
	}`

	if _, err := expandBody(body); err == nil {
		t.Fatalf("expandBody should have failed")
	}
}

func TestNewCluster(t *testing.T) {
	cluster := tester.MockContainerService("name", "westus", "dnsprefix")
	wrappedCluster := newContainerService(cluster)

	assert.Equal(t, cluster.Name, wrappedCluster.Name, "cluster names should be equal")
	assert.Equal(t, cluster.Location, wrappedCluster.Location, "cluster locations should be equal")
	assert.Equal(t, cluster.Properties.MasterProfile.DNSPrefix, wrappedCluster.Properties.MasterProfile.DNSPrefix, "cluster locations should be equal")
}
