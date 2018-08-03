package acsengine

import "testing"

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
		t.Fatalf("")
	}

	v, ok := parameters["groceries"]
	if !ok {
		t.Fatalf("could not find `groceries`")
	}
	paramsGroceries := v.(map[string]interface{})
	if len(paramsGroceries) != 2 {
		t.Fatalf("length of grocery list is not correct: expected 2 and found %d", len(paramsGroceries))
	}
	v, ok = paramsGroceries["quinoa"]
	if !ok {
		t.Fatalf("could not find `quinoa`")
	}
	item := v.(string)
	if item != "5" {
		t.Fatalf("Expected price of quinoa to be 5 but got %s", item)
	}

	v, ok = template["groceries"]
	if !ok {
		t.Fatalf("could not find `groceries`")
	}
	templateGroceries := v.(map[string]interface{})
	if len(templateGroceries) != 2 {
		t.Fatalf("length of grocery list is not correct: expected 2 and found %d", len(templateGroceries))
	}
	v, ok = templateGroceries["pasta"]
	if !ok {
		t.Fatalf("could not find `pasta`")
	}
	item = v.(string)
	if item != "2" {
		t.Fatalf("Expected price of pasta to be 2 but got %s", item)
	}
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
	if len(groceries) != 2 {
		t.Fatalf("length of grocery list is not correct: expected 2 and found %d", len(groceries))
	}
	v, ok = groceries["bananas"]
	if !ok {
		t.Fatalf("could not find `bananas`")
	}
	item := v.(string)
	if item != "5" {
		t.Fatalf("Expected price of bananas to be 5 but got %s", item)
	}
}
