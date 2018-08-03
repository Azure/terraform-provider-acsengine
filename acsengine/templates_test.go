package acsengine

import "testing"

func TestACSEngineK8sCluster_addValue(t *testing.T) {
	parameters := map[string]interface{}{}

	addValue(parameters, "key", "data")

	if v, ok := parameters["key"]; ok {
		val := v.(map[string]interface{})
		if val["value"] != "data" {
			t.Fatalf("value not set correctly")
		}
	} else {
		t.Fatalf("could not find key")
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

	if v, ok := parameters["groceries"]; ok {
		groceries := v.(map[string]interface{})
		if len(groceries) != 2 {
			t.Fatalf("length of grocery list is not correct: expected 2 and found %d", len(groceries))
		}
		if v, ok := groceries["quinoa"]; ok {
			item := v.(string)
			if item != "5" {
				t.Fatalf("Expected price of quinoa to be 5 but got %s", item)
			}
		} else {
			t.Fatalf("could not find `quinoa`")
		}
	} else {
		t.Fatalf("could not find `groceries`")
	}

	if v, ok := template["groceries"]; ok {
		groceries := v.(map[string]interface{})
		if len(groceries) != 2 {
			t.Fatalf("length of grocery list is not correct: expected 2 and found %d", len(groceries))
		}
		if v, ok := groceries["pasta"]; ok {
			item := v.(string)
			if item != "2" {
				t.Fatalf("Expected price of pasta to be 2 but got %s", item)
			}
		} else {
			t.Fatalf("could not find `pasta`")
		}
	} else {
		t.Fatalf("could not find `groceries`")
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

	if v, ok := expandedBody["groceries"]; ok {
		groceries := v.(map[string]interface{})
		if len(groceries) != 2 {
			t.Fatalf("length of grocery list is not correct: expected 2 and found %d", len(groceries))
		}
		if v, ok := groceries["bananas"]; ok {
			item := v.(string)
			if item != "5" {
				t.Fatalf("Expected price of bananas to be 5 but got %s", item)
			}
		} else {
			t.Fatalf("could not find `bananas`")
		}
	} else {
		t.Fatalf("could not find `groceries`")
	}
}
