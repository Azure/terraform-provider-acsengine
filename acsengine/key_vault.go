package acsengine

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/api"
	vaultsvc "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
)

// I ought to check if something like this exists in keyvault package already
// type keyVault struct {
// 	name    string
// 	vaultID string
// 	// secretName string
// 	// artifacts []string
// }

// certificate profile need to be set
func setKeys(c *ArmClient, cluster *Cluster) error {
	var err error
	// set them in key vault
	keyVaultURI := cluster.Properties.ServicePrincipalProfile.KeyvaultSecretRef.VaultID // I need URI not ID
	certificateProfile := cluster.Properties.CertificateProfile
	dnsPrefix := cluster.Properties.MasterProfile.DNSPrefix

	if err = setKey(c, fmt.Sprintf("%s-cacrt", dnsPrefix), keyVaultURI, certificateProfile.CaCertificate); err != nil {
		return fmt.Errorf("error setting ca certificate: %+v", err)
	}
	if err = setKey(c, fmt.Sprintf("%s-cakey", dnsPrefix), keyVaultURI, certificateProfile.CaPrivateKey); err != nil {
		return fmt.Errorf("error setting ca key: %+v", err)
	}
	if err = setKey(c, fmt.Sprintf("%s-apiservercrt", dnsPrefix), keyVaultURI, certificateProfile.APIServerCertificate); err != nil {
		return fmt.Errorf("error setting api server certificate: %+v", err)
	}
	if err = setKey(c, fmt.Sprintf("%s-apiserverkey", dnsPrefix), keyVaultURI, certificateProfile.APIServerPrivateKey); err != nil {
		return fmt.Errorf("error setting api server key: %+v", err)
	}
	if err = setKey(c, fmt.Sprintf("%s-clientcrt", dnsPrefix), keyVaultURI, certificateProfile.ClientCertificate); err != nil {
		return fmt.Errorf("error setting client certificate: %+v", err)
	}
	if err = setKey(c, fmt.Sprintf("%s-clientkey", dnsPrefix), keyVaultURI, certificateProfile.ClientPrivateKey); err != nil {
		return fmt.Errorf("error setting client key: %+v", err)
	}
	if err = setKey(c, fmt.Sprintf("%s-etcdservercrt", dnsPrefix), keyVaultURI, certificateProfile.EtcdServerCertificate); err != nil {
		return fmt.Errorf("error setting etcd server certificate: %+v", err)
	}
	if err = setKey(c, fmt.Sprintf("%s-etcdserverkey", dnsPrefix), keyVaultURI, certificateProfile.EtcdServerPrivateKey); err != nil {
		return fmt.Errorf("error setting etcd server key: %+v", err)
	}
	if err = setKey(c, fmt.Sprintf("%s-etcdclientcrt", dnsPrefix), keyVaultURI, certificateProfile.EtcdClientCertificate); err != nil {
		return fmt.Errorf("error setting etcd client certificate: %+v", err)
	}
	if err = setKey(c, fmt.Sprintf("%s-etcdclientkey", dnsPrefix), keyVaultURI, certificateProfile.EtcdClientPrivateKey); err != nil {
		return fmt.Errorf("error setting etcd client key: %+v", err)
	}
	for i, crt := range certificateProfile.EtcdPeerCertificates {
		if err = setKey(c, keyVaultURI, fmt.Sprintf("%s-etcdpeer%dcrt", dnsPrefix, i), crt); err != nil {
			return fmt.Errorf("error setting etcdpeer%d certificate: %+v", i, err)
		}
	}
	for i, key := range certificateProfile.EtcdPeerPrivateKeys {
		if err = setKey(c, keyVaultURI, fmt.Sprintf("%s-etcdpeer%dkey", dnsPrefix, i), key); err != nil {
			return fmt.Errorf("error setting etcdpeer%d key: %+v", i, err)
		}
	}

	// also set azuredeploy file to only have vault uri (I probably shouldn't call WriteTLSArtifacts until after)

	return nil
}

func setKey(c *ArmClient, vaultID, name, value string) error {
	parameters := vaultsvc.SecretSetParameters{
		Value: &value,
	}
	_, err := c.keyVaultManagementClient.SetSecret(c.StopContext, vaultID, name, parameters)
	if err != nil {
		return fmt.Errorf("failed to get secret: %+v", err)
	}
	return nil
}

func getKey(c *ArmClient, ref *api.KeyvaultSecretRef) (string, error) {
	keyVaultClient := c.keyVaultClient
	keyVaultManagementClient := c.keyVaultManagementClient

	vaultID := ref.VaultID // I need URI not ID
	secretName := ref.SecretName
	// version := sp.KeyvaultSecretRef.SecretVersion

	id, err := utils.ParseAzureResourceID(vaultID)
	if err != nil {
		return "", err
	}
	name, ok := id.Path["vaults"]
	if !ok {
		return "", fmt.Errorf("could not find vault name")
	}
	resp, err := keyVaultClient.Get(c.StopContext, id.ResourceGroup, name)
	if err != nil {
		return "", fmt.Errorf("failed to get key vault: %+v", err)
	}

	props := resp.Properties
	if props == nil {
		return "", fmt.Errorf("properties not found")
	}
	vaultURI := props.VaultURI
	if vaultURI == nil {
		return "", fmt.Errorf("vault uri not found")
	}

	read, err := keyVaultManagementClient.GetSecret(c.StopContext, *vaultURI, secretName, "")
	if err != nil {
		return "", fmt.Errorf("error reading key vault %s", vaultID)
	}
	if read.ID == nil {
		return "", fmt.Errorf("cannot read key vault secret %s (in key vault %s)", secretName, vaultID)
	}
	if read.Value == nil {
		return "", fmt.Errorf("key value is not set")
	}
	fmt.Println(read.Value)
	return *read.Value, nil
}
