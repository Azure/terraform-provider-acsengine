package utils

import (
	"strings"
	"testing"
)

func TestClusterKubeConfig(t *testing.T) {
	dnsPrefix := "prefix"
	location := "southcentralus"
	fqdn := "https://prefix.southcentralus.cloudapp.azure.com"
	config := ACSEngineK8sClusterKubeConfig(dnsPrefix, location)

	if !strings.Contains(config, fqdn) {
		t.Fatalf("expected kube config to contain %s", fqdn)
	}
}
