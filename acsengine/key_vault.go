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
func setCertificateProfileSecrets(c *ArmClient, cluster *Cluster) error {
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

	if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-cacrt", dnsPrefix), certificateProfile.CaCertificate); err != nil {
		return fmt.Errorf("error setting ca certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-cakey", dnsPrefix), certificateProfile.CaPrivateKey); err != nil {
		return fmt.Errorf("error setting ca key: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-apiservercrt", dnsPrefix), certificateProfile.APIServerCertificate); err != nil {
		return fmt.Errorf("error setting api server certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-apiserverkey", dnsPrefix), certificateProfile.APIServerPrivateKey); err != nil {
		return fmt.Errorf("error setting api server key: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-clientcrt", dnsPrefix), certificateProfile.ClientCertificate); err != nil {
		return fmt.Errorf("error setting client certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-clientkey", dnsPrefix), certificateProfile.ClientPrivateKey); err != nil {
		return fmt.Errorf("error setting client key: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-etcdservercrt", dnsPrefix), certificateProfile.EtcdServerCertificate); err != nil {
		return fmt.Errorf("error setting etcd server certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-etcdserverkey", dnsPrefix), certificateProfile.EtcdServerPrivateKey); err != nil {
		return fmt.Errorf("error setting etcd server key: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-etcdclientcrt", dnsPrefix), certificateProfile.EtcdClientCertificate); err != nil {
		return fmt.Errorf("error setting etcd client certificate: %+v", err)
	}
	if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-etcdclientkey", dnsPrefix), certificateProfile.EtcdClientPrivateKey); err != nil {
		return fmt.Errorf("error setting etcd client key: %+v", err)
	}
	for i, crt := range certificateProfile.EtcdPeerCertificates {
		if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-etcdpeer%dcrt", dnsPrefix, i), crt); err != nil {
			return fmt.Errorf("error setting etcdpeer%d certificate: %+v", i, err)
		}
	}
	for i, key := range certificateProfile.EtcdPeerPrivateKeys {
		if err = setSecret(c, keyVaultURI, fmt.Sprintf("%s-etcdpeer%dkey", dnsPrefix, i), key); err != nil {
			return fmt.Errorf("error setting etcdpeer%d key: %+v", i, err)
		}
	}

	// also set azuredeploy file to only have vault uri (I probably shouldn't call WriteTLSArtifacts until after)

	return nil
}

// why am I able to get but not set secrets?
func setSecret(c *ArmClient, vaultURI, name, value string) error {
	parameters := vaultsvc.SecretSetParameters{
		Value: &value,
		// ContentType:
	}
	_, err := c.keyVaultManagementClient.SetSecret(c.StopContext, vaultURI, name, parameters)
	if err != nil {
		return fmt.Errorf("failed to set secret: %+v", err)
	}
	return nil
}

func getSecret(c *ArmClient, vaultID, secretName, version string) (string, error) {
	keyVaultManagementClient := c.keyVaultManagementClient

	// I need URI not ID
	// version := sp.KeyvaultSecretRef.SecretVersion

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

	read, err := keyVaultManagementClient.GetSecret(c.StopContext, *vaultURI, secretName, "")
	if err != nil {
		return "", fmt.Errorf("error reading key vault %s with vault URI %s: %+v", vaultID, *vaultURI, err)
	}
	if read.ID == nil {
		return "", fmt.Errorf("cannot read key vault secret %s (in key vault %s) with URI %s", secretName, vaultID, *vaultURI)
	}
	if read.Value == nil {
		return "", fmt.Errorf("key value is not set")
	}
	fmt.Println(read.Value)
	return *read.Value, nil
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

// if it turns out this is a problem with acs-engine then I only need the second function, which will generate the parameters correctly

// func setCertificateProfileSecretsParameters(c *ArmClient, cluster *Cluster, params string) error {
// 	certificateProfile := cluster.Properties.CertificateProfile
// 	vaultID := cluster.Properties.ServicePrincipalProfile.KeyvaultSecretRef.VaultID
// 	dnsPrefix := cluster.Properties.MasterProfile.DNSPrefix

// 	parametersMap, err := expandBody(params)
// 	if err != nil {
// 		return fmt.Errorf("failed to expand parameters string: %+v", err)
// 	}

// 	v, ok := parametersMap["parameters"]
// 	if !ok {
// 		return fmt.Errorf("parameters not formatted correctly")
// 	}
// 	parameters := v.(map[string]interface{})

// 	// this actually has to be a reference block
// 	// "reference": {
// 	// 	"keyVault": {
// 	// 	  "id": "/subscriptions/<SUB_ID>/resourceGroups/<RG_NAME>/providers/Microsoft.KeyVault/vaults/<KV_NAME>"
// 	// 	},
// 	// 	"secretName": "<NAME>"
// 	// 	"secretVersion": "<VERSION>"
// 	//  }
// 	// parameters["caCertificate"] = fmt.Sprintf("%s/secrets/%s-cacert", vaultID, dnsPrefix)
// 	// parameters["caPrivateKey"] = fmt.Sprintf("%s/secrets/%s-cakey", vaultID, dnsPrefix)
// 	// parameters["apiServerCertificate"] = fmt.Sprintf("%s/secrets/%s-apiservercrt", vaultID, dnsPrefix)
// 	// parameters["apiServerPrivateKey"] = fmt.Sprintf("%s/secrets/%s-apiserverkey", vaultID, dnsPrefix)
// 	// parameters["clientCertificate"] = fmt.Sprintf("%s/secrets/%s-clientcrt", vaultID, dnsPrefix)
// 	// parameters["clientPrivateKey"] = fmt.Sprintf("%s/secrets/%s-clientkey", vaultID, dnsPrefix)
// 	// parameters["kubeConfigCertificate"] = fmt.Sprintf("%s/secrets/%s-kubeconfigcrt", vaultID, dnsPrefix)
// 	// parameters["kubeConfigPrivateKey"] = fmt.Sprintf("%s/secrets/%s-kubeconfigkey", vaultID, dnsPrefix)
// 	// parameters["etcdClientCertificate"] = fmt.Sprintf("%s/secrets/%s-etcdclientcrt", vaultID, dnsPrefix)
// 	// parameters["etcdClientPrivateKey"] = fmt.Sprintf("%s/secrets/%s-etcdclientkey", vaultID, dnsPrefix)
// 	// parameters["etcdServerCertificate"] = fmt.Sprintf("%s/secrets/%s-etcdservercrt", vaultID, dnsPrefix)
// 	// parameters["etcdServerPrivateKey"] = fmt.Sprintf("%s/secrets/%s-etcdserverkey", vaultID, dnsPrefix)

// 	for i := range certificateProfile.EtcdPeerCertificates {
// 		parameters[fmt.Sprintf("etcdPeerCertificate%d", i)] = fmt.Sprintf("%s/secrets/%s-etcdpeer%dcrt", vaultID, dnsPrefix, i)
// 	}
// 	for i := range certificateProfile.EtcdClientPrivateKey {
// 		parameters[fmt.Sprintf("etcdPeerPrivateKey%d", i)] = fmt.Sprintf("%s/secrets/%s-etcdpeer%dkey", vaultID, dnsPrefix, i)
// 	}

// 	return nil
// }

// func setCertificateProfileSecretsAPIModel(c *ArmClient, cluster *Cluster, params string) error {
// 	certificateProfile := cluster.Properties.CertificateProfile
// 	vaultID := cluster.Properties.ServicePrincipalProfile.KeyvaultSecretRef.VaultID
// 	dnsPrefix := cluster.Properties.MasterProfile.DNSPrefix

// 	certificateProfile.CaCertificate = fmt.Sprintf("%s/secrets/%s-cacert", vaultID, dnsPrefix)
// 	certificateProfile.CaPrivateKey = fmt.Sprintf("%s/secrets/%s-cakey", vaultID, dnsPrefix)
// 	certificateProfile.APIServerCertificate = fmt.Sprintf("%s/secrets/%s-apiservercrt", vaultID, dnsPrefix)
// 	certificateProfile.APIServerPrivateKey = fmt.Sprintf("%s/secrets/%s-apiserverkey", vaultID, dnsPrefix)
// 	certificateProfile.ClientCertificate = fmt.Sprintf("%s/secrets/%s-clientcrt", vaultID, dnsPrefix)
// 	certificateProfile.ClientPrivateKey = fmt.Sprintf("%s/secrets/%s-clientkey", vaultID, dnsPrefix)
// 	certificateProfile.KubeConfigCertificate = fmt.Sprintf("%s/secrets/%s-kubeconfigcrt", vaultID, dnsPrefix)
// 	certificateProfile.KubeConfigPrivateKey = fmt.Sprintf("%s/secrets/%s-kubeconfigkey", vaultID, dnsPrefix)
// 	certificateProfile.EtcdClientCertificate = fmt.Sprintf("%s/secrets/%s-etcdclientcrt", vaultID, dnsPrefix)
// 	certificateProfile.EtcdClientPrivateKey = fmt.Sprintf("%s/secrets/%s-etcdclientcrt", vaultID, dnsPrefix)
// 	certificateProfile.EtcdServerCertificate = fmt.Sprintf("%s/secrets/%s-etcdservercrt", vaultID, dnsPrefix)
// 	certificateProfile.EtcdClientPrivateKey = fmt.Sprintf("%s/secrets/%s-etcdserverkey", vaultID, dnsPrefix)

// 	for i := range certificateProfile.EtcdPeerCertificates {
// 		certificateProfile.EtcdPeerCertificates[i] = fmt.Sprintf("%s/secrets/%s-etcdpeer%dcrt", vaultID, dnsPrefix, i)
// 	}
// 	for i := range certificateProfile.EtcdPeerCertificates {
// 		certificateProfile.EtcdPeerPrivateKeys[i] = fmt.Sprintf("%s/secrets/%s-etcdpeer%dkey", vaultID, dnsPrefix, i)
// 	}

// 	return nil
// }
