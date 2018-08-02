package acsengine

import (
	"encoding/json"
	"fmt"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/acsengine/transform"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/i18n"
)

func addValue(params map[string]interface{}, k string, v interface{}) {
	params[k] = map[string]interface{}{
		"value": v,
	}
}

func expandTemplates(template string, parameters string) (map[string]interface{}, map[string]interface{}, error) {
	templateBody, err := expandBody(template)
	if err != nil {
		return nil, nil, fmt.Errorf("error expanding the template_body for Azure RM Template Deployment: %+v", err)
	}
	parametersBody, err := expandBody(parameters)
	if err != nil {
		return nil, nil, fmt.Errorf("error expanding the parameters_body for Azure RM Template Deployment: %+v", err)
	}
	return templateBody, parametersBody, nil
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

func formatTemplates(cluster *api.ContainerService) (string, string, bool, error) {
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return "", "", false, fmt.Errorf("error loading translations: %+v", err)
	}
	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}

	templateGenerator, err := acsengine.InitializeTemplateGenerator(ctx, false)
	if err != nil {
		return "", "", false, fmt.Errorf("failed to initialize template generator: %+v", err)
	}
	template, parameters, certsGenerated, err := templateGenerator.GenerateTemplate(cluster, acsengine.DefaultGeneratorCode, false, false, acsEngineVersion)
	if err != nil {
		return "", "", false, fmt.Errorf("error generating templates: %+v", err)
	}

	template, err = transform.PrettyPrintArmTemplate(template)
	if err != nil {
		return "", "", false, fmt.Errorf("error pretty printing template: %+v", err)
	}
	parameters, err = transform.BuildAzureParametersFile(parameters)
	if err != nil {
		return "", "", false, fmt.Errorf("error pretty printing template parameters: %+v", err)
	}

	return template, parameters, certsGenerated, nil
}
