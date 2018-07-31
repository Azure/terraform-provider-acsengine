package acsengine

import (
	"encoding/json"
	"fmt"
)

func addValue(params map[string]interface{}, k string, v interface{}) {
	params[k] = map[string]interface{}{
		"value": v,
	}
}

func expandTemplateBody(template string) (map[string]interface{}, error) {
	templateBody, err := expandBody(template)
	if err != nil {
		return nil, fmt.Errorf("error expanding the template_body for Azure RM Template Deployment: %+v", err)
	}
	return templateBody, nil
}

func expandParametersBody(parameters string) (map[string]interface{}, error) {
	parametersBody, err := expandBody(parameters)
	if err != nil {
		return nil, fmt.Errorf("error expanding the parameters_body for Azure RM Template Deployment: %+v", err)
	}
	return parametersBody, nil
}

func expandBody(body string) (map[string]interface{}, error) {
	var bodyMap map[string]interface{}
	if err := json.Unmarshal([]byte(body), &bodyMap); err != nil {
		return nil, err
	}
	return bodyMap, nil
}
