package acsengine

import (
	"fmt"
	"testing"

	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/terraform/terraform"
)

func TestSetUserAgent(t *testing.T) {
	client := &autorest.Client{}

	setUserAgent(client)

	tfVersion := fmt.Sprintf("HashiCorp-Terraform-v%s", terraform.VersionString())
	if client.UserAgent != tfVersion {
		t.Fatalf("client.UserAgent- actual: %s, expected: %s", client.UserAgent, tfVersion)
	}
}
