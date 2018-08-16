package acsengine

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/mgmt/keyvault"
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
func setCertificateProfileSecretsKeyVault(c *ArmClient, cluster *Cluster) error {
	var err error
	// set them in key vault
	keyVaultID := cluster.Properties.ServicePrincipalProfile.KeyvaultSecretRef.VaultID // I need URI not ID
	certificateProfile := cluster.Properties.CertificateProfile
	dnsPrefix := cluster.Properties.MasterProfile.DNSPrefix

	resp, err := getKeyVault(c, keyVaultID)
	if err != nil {
		return fmt.Errorf("failed to get key vault: %+v", err)
	}
	keyVaultURI := *resp.Properties.VaultURI

	if err = setSecret(c, keyVaultURI, secretName("cacrt", dnsPrefix), base64Encode(certificateProfile.CaCertificate)); err != nil {
		return fmt.Errorf("error setting ca certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("cakey", dnsPrefix), base64Encode(certificateProfile.CaPrivateKey)); err != nil {
		return fmt.Errorf("error setting ca key: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("apiservercrt", dnsPrefix), base64Encode(certificateProfile.APIServerCertificate)); err != nil {
		return fmt.Errorf("error setting api server certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("apiserverkey", dnsPrefix), base64Encode(certificateProfile.APIServerPrivateKey)); err != nil {
		return fmt.Errorf("error setting api server key: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("clientcrt", dnsPrefix), base64Encode(certificateProfile.ClientCertificate)); err != nil {
		return fmt.Errorf("error setting client certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("clientkey", dnsPrefix), base64Encode(certificateProfile.ClientPrivateKey)); err != nil {
		return fmt.Errorf("error setting client key: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("kubeconfigcrt", dnsPrefix), base64Encode(certificateProfile.KubeConfigCertificate)); err != nil {
		return fmt.Errorf("error setting client certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("kubeconfigkey", dnsPrefix), base64Encode(certificateProfile.KubeConfigPrivateKey)); err != nil {
		return fmt.Errorf("error setting client key: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("etcdservercrt", dnsPrefix), base64Encode(certificateProfile.EtcdServerCertificate)); err != nil {
		return fmt.Errorf("error setting etcd server certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("etcdserverkey", dnsPrefix), base64Encode(certificateProfile.EtcdServerPrivateKey)); err != nil {
		return fmt.Errorf("error setting etcd server key: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("etcdclientcrt", dnsPrefix), base64Encode(certificateProfile.EtcdClientCertificate)); err != nil {
		return fmt.Errorf("error setting etcd client certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, secretName("etcdclientkey", dnsPrefix), base64Encode(certificateProfile.EtcdClientPrivateKey)); err != nil {
		return fmt.Errorf("error setting etcd client key: %+v", err)
	}
	for i, crt := range certificateProfile.EtcdPeerCertificates {
		if err = setSecret(c, keyVaultURI, secretName(fmt.Sprintf("etcdpeer%dcrt", i), dnsPrefix), base64Encode(crt)); err != nil {
			return fmt.Errorf("error setting etcdpeer%d certificate: %+v", i, err)
		}
	}
	for i, key := range certificateProfile.EtcdPeerPrivateKeys {
		if err = setSecret(c, keyVaultURI, secretName(fmt.Sprintf("etcdpeer%dkey", i), dnsPrefix), base64Encode(key)); err != nil {
			return fmt.Errorf("error setting etcdpeer%d key: %+v", i, err)
		}
	}

	return nil
}

// for setting kube config correctly
func getCertificateProfileSecretsKeyVault(c *ArmClient, cluster *Cluster) error {
	vaultID := cluster.Properties.ServicePrincipalProfile.KeyvaultSecretRef.VaultID
	dnsPrefix := cluster.Properties.MasterProfile.DNSPrefix

	var val string

	resp, err := getKeyVault(c, vaultID)
	if err != nil {
		return fmt.Errorf("failed to get key vault: %+v", err)
	}

	props := resp.Properties
	if props == nil {
		return fmt.Errorf("properties not found")
	}
	vaultURI := props.VaultURI
	if vaultURI == nil {
		return fmt.Errorf("vault uri not found")
	}

	fmt.Printf("key vault: %s\n", *vaultURI)

	if val, err = getSecret(c, *vaultURI, secretName("cacrt", dnsPrefix), ""); err != nil {
		return fmt.Errorf("failed to get ca.crt")
	}
	cluster.Properties.CertificateProfile.CaCertificate = base64Decode(val)
	if val, err = getSecret(c, *vaultURI, secretName("kubeconfigcrt", dnsPrefix), ""); err != nil {
		return fmt.Errorf("failed to get kubectlClient.crt")
	}
	cluster.Properties.CertificateProfile.KubeConfigCertificate = base64Decode(val)
	if val, err = getSecret(c, *vaultURI, secretName("kubeconfigkey", dnsPrefix), ""); err != nil {
		return fmt.Errorf("failed to get kubectlClient.key")
	}
	cluster.Properties.CertificateProfile.KubeConfigPrivateKey = base64Decode(val)

	return nil
}

func setSecret(c *ArmClient, vaultURI, name, value string) error {
	contentType := "base64" // is this valid?
	parameters := vaultsvc.SecretSetParameters{
		Value:       &value,
		ContentType: &contentType,
	}
	_, err := c.keyVaultManagementClient.SetSecret(c.StopContext, vaultURI, name, parameters)
	if err != nil {
		return fmt.Errorf("failed to set secret: %+v", err)
	}
	return nil
}

func getSecret(c *ArmClient, vaultURI, name, version string) (string, error) {
	keyVaultManagementClient := c.keyVaultManagementClient

	read, err := keyVaultManagementClient.GetSecret(c.StopContext, vaultURI, name, version)
	if err != nil {
		return "", fmt.Errorf("error reading key vault with vault URI %s: %+v", vaultURI, err)
	}
	if read.ID == nil {
		return "", fmt.Errorf("cannot read key vault secret %s with key vault URI %s", name, vaultURI)
	}
	if read.Value == nil {
		return "", fmt.Errorf("key value is not set")
	}
	return *read.Value, nil
}

func getSecretFromKeyVault(c *ArmClient, vaultID, secretName, version string) (string, error) {
	resp, err := getKeyVault(c, vaultID)
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

	fmt.Printf("key vault: %s\n", *vaultURI)

	return getSecret(c, *vaultURI, secretName, version)
}

func getKeyVault(c *ArmClient, vaultID string) (*keyvault.Vault, error) {
	keyVaultClient := c.keyVaultClient

	id, err := utils.ParseAzureResourceID(vaultID)
	if err != nil {
		return nil, err
	}
	name, ok := id.Path["vaults"]
	if !ok {
		return nil, fmt.Errorf("could not find vault name")
	}
	resp, err := keyVaultClient.Get(c.StopContext, id.ResourceGroup, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get key vault: %+v", err)
	}
	return &resp, nil
}

func (cluster *Cluster) setCertificateProfileSecretsAPIModel() error {
	certificateProfile := cluster.Properties.CertificateProfile
	vaultID := cluster.Properties.ServicePrincipalProfile.KeyvaultSecretRef.VaultID
	dnsPrefix := cluster.Properties.MasterProfile.DNSPrefix

	certificateProfile.CaCertificate = vaultSecretRefName("cacrt", vaultID, dnsPrefix)
	certificateProfile.CaPrivateKey = vaultSecretRefName("cakey", vaultID, dnsPrefix)
	certificateProfile.APIServerCertificate = vaultSecretRefName("apiservercrt", vaultID, dnsPrefix)
	certificateProfile.APIServerPrivateKey = vaultSecretRefName("apiserverkey", vaultID, dnsPrefix)
	certificateProfile.ClientCertificate = vaultSecretRefName("clientcrt", vaultID, dnsPrefix)
	certificateProfile.ClientPrivateKey = vaultSecretRefName("clientkey", vaultID, dnsPrefix)
	certificateProfile.KubeConfigCertificate = vaultSecretRefName("kubeconfigcrt", vaultID, dnsPrefix)
	certificateProfile.KubeConfigPrivateKey = vaultSecretRefName("kubeconfigkey", vaultID, dnsPrefix)
	certificateProfile.EtcdClientCertificate = vaultSecretRefName("etcdclientcrt", vaultID, dnsPrefix)
	certificateProfile.EtcdClientPrivateKey = vaultSecretRefName("etcdclientkey", vaultID, dnsPrefix)
	certificateProfile.EtcdServerCertificate = vaultSecretRefName("etcdservercrt", vaultID, dnsPrefix)
	certificateProfile.EtcdServerPrivateKey = vaultSecretRefName("etcdserverkey", vaultID, dnsPrefix)

	for i := range certificateProfile.EtcdPeerCertificates {
		certificateProfile.EtcdPeerCertificates[i] = vaultSecretRefName(fmt.Sprintf("etcdpeer%dcrt", i), vaultID, dnsPrefix)
	}
	for i := range certificateProfile.EtcdPeerCertificates {
		certificateProfile.EtcdPeerPrivateKeys[i] = vaultSecretRefName(fmt.Sprintf("etcdpeer%dkey", i), vaultID, dnsPrefix)
	}

	return nil
}

func secretName(name, dnsPrefix string) string {
	return fmt.Sprintf("%s-%s", dnsPrefix, name)
}

func vaultSecretRefName(name, vaultID, dnsPrefix string) string {
	return fmt.Sprintf("%s/secrets/%s", vaultID, secretName(name, dnsPrefix))
}
