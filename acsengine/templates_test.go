package acsengine

import (
	"testing"

	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/test"
)

func TestACSEngineK8sCluster_addValue(t *testing.T) {
	parameters := map[string]interface{}{}

	addValue(parameters, "key", "data")

	v, ok := parameters["key"]
	if !ok {
		t.Fatalf("could not find key")
	}
	val := v.(map[string]interface{})
	if val["value"] != "data" {
		t.Fatalf("value not set correctly")
	}
}

func TestACSEngineK8sCluster_expandTemplateBodies(t *testing.T) {
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
	if !ok {
		t.Fatalf("could not find `groceries`")
	}
	paramsGroceries := v.(map[string]interface{})
	if len(paramsGroceries) != 2 {
		t.Fatalf("length of grocery list is not correct: expected 2 and found %d", len(paramsGroceries))
	}
	test.Equals(t, len(paramsGroceries), 2)
	v, ok = paramsGroceries["quinoa"]
	if !ok {
		t.Fatalf("could not find `quinoa`")
	}
	test.Equals(t, v.(string), "5")

	v, ok = template["groceries"]
	if !ok {
		t.Fatalf("could not find `groceries`")
	}
	templateGroceries := v.(map[string]interface{})
	if len(templateGroceries) != 2 {
		t.Fatalf("length of grocery list is not correct: expected 2 and found %d", len(templateGroceries))
	}
	test.Equals(t, len(templateGroceries), 2)
	v, ok = templateGroceries["pasta"]
	if !ok {
		t.Fatalf("could not find `pasta`")
	}
	test.Equals(t, v.(string), "2")
}

func TestACSEngineK8sCluster_expandBody(t *testing.T) {
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
	if !ok {
		t.Fatalf("could not find `groceries`")
	}
	groceries := v.(map[string]interface{})
	test.Equals(t, len(groceries), 2)
	v, ok = groceries["bananas"]
	if !ok {
		t.Fatalf("could not find `bananas`")
	}
	test.Equals(t, v.(string), "5")
}
