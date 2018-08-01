package acsengine

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/i18n"
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

func writeTemplatesAndCerts(cluster *api.ContainerService, template string, parameters string, deploymentDirectory string, certsGenerated bool) error {
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return fmt.Errorf("error loading translations: %+v", err)
	}

	// save templates and certificates
	writer := &acsengine.ArtifactWriter{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}
	if err = writer.WriteTLSArtifacts(cluster, apiVersion, template, parameters, deploymentDirectory, certsGenerated, false); err != nil {
		return fmt.Errorf("error writing artifacts: %+v", err)
	}

	return nil
}
