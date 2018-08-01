package acsengine

import "testing"

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
