# Importing Resources

ACS Engine clusters can be imported using the deployment resource ID and the directory containing their apimodel.json file delimited by a space.

For example, if the resource ID of your cluster deployment is "/subscriptions/1234/resourceGroups/testrg/providers/Microsoft.Resources/deployments/deploymentName" and the directory containing `apimodel.json` is "_output/dnsPrefix", then the import command will be:

```terraform import acsengine_kubernetes_cluster.example "/subscriptions/1234/resourceGroups/testrg/providers/Microsoft.Resources/deployments/deploymentName _output/dnsPrefix"```

Remember to surround the ID string by quotes since there is a space. Also, do not forget to add properties to `apimodel.json` that you specified on the command line when deploying acs-engine templates. For instance, add `location` and `name` (i.e. deployment name). You can add these to `apimodel.json`. If you are unsure what the apimodel should look like, you can take a look at an 'apimodel.json' file created with this Terraform provider (or output the `api_model` field from an existing resource, decoded from base64).

You will also need to add key vault secret references to this file if you are not already using them, and make sure they follow the naming convention mentioned below. You will need to add a reference for your service principal secret and for all of your certificates and keys. You will need to set your secrets in the same key vault where you are storing your service principal client ID. For information on correct formatting, look at this [ACS-Engine doc](https://github.com/Azure/acs-engine/tree/master/examples/keyvault-params).

You must store your keys with the following format:

```masterdnsprefix-cacrt```

where "masterdnsprefix" is your master DNS prefix, and "cacrt" is the name of the certificate without a space. The certificate names are as follows:

apiServerCertificate: `apiservercrt`
apiServerPrivateKey: `apiserverkey`
caCertificate: `cacrt`
caPrivateKey: `cakey`
clientCertificate: `clientcrt`
clientPrivateKey: `clientkey`
kubeConfigCertificate: `kubectlcrt`
kubeConfigPrivateKey: `kubectlkey`
etcdServerCertificate: `etcdservercrt`
etcdServerPrivateKey: `etcdserverkey`
etcdClientCertificate: `etcdclientcrt`
etcdClientPrivateKey: `etcdclientkey`
etcdPeerCertificates: `etcdpeer0crt`, where `0` is replaces by the number of the etcd peer certificate.
etcdPeerPrivateKeys: `etcdpeer0key`, where `0` is replaced by the number of the etcd peer key.

Here is an example script for setting secrets using Azure CLI, from the deployment directory created by generating acs-engine templates. You can name your service principal secret whatever you want as long as you include it in the Terraform configuration.

```bash
#!/bin/bash

az keyvault secret set --vault-name <KV_NAME> --name spsecret --value $ARM_CLIENT_SECRET

az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-cacrt --value $(cat ca.crt | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-cakey --value $(cat ca.key | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-apiservercrt --value $(cat apiserver.crt | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-apiserverkey --value $(cat apiserver.key | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-clientcrt --value $(cat client.crt | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-clientkey --value $(cat client.key | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-kubectlClientcrt --value $(cat kubectlClient.crt | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-kubectlClientkey --value $(cat kubectlClient.key | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-etcdclientcrt --value $(cat etcdclient.crt | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-etcdclientkey --value $(cat etcdclient.key | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-etcdservercrt --value $(cat etcdserver.crt | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-etcdserverkey --value $(cat etcdserver.key | base64 --wrap=0)

az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-etcdpeer0crt --value $(cat etcdpeer0.crt | base64 --wrap=0)
az keyvault secret set --vault-name <KV_NAME> --name <DNS_PREFIX>-etcdpeer0key --value $(cat etcdpeer0.key | base64 --wrap=0)
```

This may seem like a lot to change, but if `terraform import` fails you can just delete your `terraform.tfstate` file, fix your problem, and try again.