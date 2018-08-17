package acsengine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/acsengine/transform"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/i18n"
)

type containerService struct {
	*api.ContainerService

	ResourceGroup    string
	ServicePrincipal string
}

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

func newContainerService(cluster *api.ContainerService) *containerService {
	return &containerService{
		ContainerService: cluster,
	}
}

func (cluster *containerService) writeTemplatesAndCerts(template string, parameters string, deploymentDirectory string, certsGenerated bool) error {
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
	if err = writer.WriteTLSArtifacts(cluster.ContainerService, apiVersion, template, parameters, deploymentDirectory, certsGenerated, false); err != nil {
		return fmt.Errorf("error writing artifacts: %+v", err)
	}

	return nil
}

func (cluster *containerService) formatTemplates(buildParamsFile bool) (string, string, bool, error) {
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
	template, parameters, certsGenerated, err := templateGenerator.GenerateTemplate(cluster.ContainerService, acsengine.DefaultGeneratorCode, false, false, acsEngineVersion)
	if err != nil {
		return "", "", false, fmt.Errorf("error generating templates: %+v", err)
	}

	if template, err = transform.PrettyPrintArmTemplate(template); err != nil {
		return "", "", false, fmt.Errorf("error pretty printing template: %+v", err)
	}
	if buildParamsFile {
		if parameters, err = transform.BuildAzureParametersFile(parameters); err != nil {
			return "", "", false, fmt.Errorf("error pretty printing template parameters: %+v", err)
		}
	}

	return template, parameters, certsGenerated, nil
}

func (cluster *containerService) saveTemplates(d *resourceData, deploymentDirectory string) error {
	template, parameters, certsGenerated, err := cluster.formatTemplates(true)
	if err != nil {
		return fmt.Errorf("failed to format templates: %+v", err)
	}

	if err = cluster.writeTemplatesAndCerts(template, parameters, deploymentDirectory, certsGenerated); err != nil {
		return fmt.Errorf("error writing templates and certificates: %+v", err)
	}
	if err = d.setStateAPIModel(cluster); err != nil {
		return fmt.Errorf("error setting API model: %+v", err)
	}

	return nil
}

func getAPIModelFromFile(deploymentDirectory string) (string, error) {
	APIModelPath := path.Join(deploymentDirectory, "apimodel.json")
	if _, err := os.Stat(APIModelPath); os.IsNotExist(err) {
		return "", fmt.Errorf("specified api model does not exist (%s)", APIModelPath)
	}
	f, err := os.Open(APIModelPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %+v", err)
	}
	defer func() { // weirdness is because I need to check return value for linter
		err := f.Close()
		if err != nil {
			log.Fatalf("error closing file: %+v", err)
		}
	}()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %+v", err)
	}
	apimodel := base64Encode(string(b))

	return apimodel, nil
}
