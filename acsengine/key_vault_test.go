package acsengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// I need to write tests

func TestSecretName(t *testing.T) {
	cases := []struct {
		name      string
		dnsPrefix string
		expected  string
	}{
		{
			name:      "cacrt",
			dnsPrefix: "prefix",
			expected:  "prefix-cacrt",
		},
	}

	for _, tc := range cases {
		name := secretName(tc.name, tc.dnsPrefix)

		assert.Equal(t, tc.expected, name, "secret name not set correctly")
	}
}

func TestVaultSecretRefName(t *testing.T) {
	cases := []struct {
		name      string
		vaultID   string
		dnsPrefix string
		expected  string
	}{
		{
			name:      "cacrt",
			vaultID:   "/subscriptions/subid/resourceGroups/rgname/providers/Microsoft.KeyVault/vaults/vaultname",
			dnsPrefix: "prefix",
			expected:  "/subscriptions/subid/resourceGroups/rgname/providers/Microsoft.KeyVault/vaults/vaultname/secrets/prefix-cacrt",
		},
	}

	for _, tc := range cases {
		name := vaultSecretRefName(tc.name, tc.vaultID, tc.dnsPrefix)

		assert.Equal(t, tc.expected, name, "secret name not set correctly")
	}
}

func TestSetCertificateProfileSecretsAPIModel(t *testing.T) {
	// certificateProfile := api.CertificateProfile
}
