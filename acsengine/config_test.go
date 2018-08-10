package acsengine

import (
	"fmt"
	"testing"

	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/terraform/terraform"
	"github.com/stretchr/testify/assert"
)

func TestSetUserAgent(t *testing.T) {
	userAgent := "userAgent"
	tfVersion := fmt.Sprintf("%s;HashiCorp-Terraform-v%s", userAgent, terraform.VersionString())
	client := &autorest.Client{
		UserAgent: userAgent,
	}

	setUserAgent(client)
	assert.Equal(t, tfVersion, client.UserAgent, "client.UserAgent value is incorrect")
}

func TestSetEmptyUserAgent(t *testing.T) {
	client := &autorest.Client{}

	setUserAgent(client)

	tfVersion := fmt.Sprintf("HashiCorp-Terraform-v%s", terraform.VersionString())
	assert.Equal(t, tfVersion, client.UserAgent, "client.UserAgent value is incorrect")
}
