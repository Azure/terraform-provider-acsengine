# acsengine_kubernetes_cluster

Manages ACS Engine Cluster

**Note:** All arguments including the client secret will be stored in the raw state as plain-text.

**Another Note:** A resource group is created for the cluster and will be destroyed when the cluster is destroyed. Do not put resources in this group that should not be deleted with the cluster.

## Example Usage

You should have an existing Azure key vault where you can store your service principal secret and cluster certificates and keys. You should store the value of the service principal secret in your key vault before creating this Terraform resource. You can use the `azurerm` Terraform provider to create resource groups, key vaults, access policies, and key vault secrets.

```hcl
data "azurerm_resource_group" "testkvrg" {
  name = "testkv"
}

data "azurerm_key_vault" "testkv" {
  name = "testkvrg"
  resource_group_name = "${data.azurerm_resource_group.testkvrg.name}"
}

data "azurerm_key_vault_secret" "spsecret" {
  name = "spsecret"
  vault_uri = "${data.azurerm_key_vault.testkv.vault_uri}"
}

resource "acsengine_kubernetes_cluster" "test" {
    name               = "testcluster"
    resource_group     = "testrg"
    location           = "southcentralus"
    kubernetes_version = "1.10.4"

    master_profile {
        count           = 1
        dns_name_prefix = "creativeDNSPrefix"
    }

    agent_pool_profiles {
        name    = "agentpool1"
        count   = 2
        vm_size = "Standard_D2_v2"
    }

    agent_pool_profiles {
        name    = "agentpool2"
        count   = 1
        vm_size = "Standard_D2_v2"
        os_disk_size = 200
    }

    linux_profile {
        admin_username = "azureuser"
        ssh {
            key_data = "ssh-rsa AAAAB3NzaC... terraform@demo.tld"
        }
    }

    service_principal {
        client_id     = ""
        vault_id      = ""
        secret_name   = ""
    }

    tags {
        Environment = "Production"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the cluster to create, which will be the deployment name. Changing this forces a new resource to be created.
* `resource_group` - (Required) Specifies the name of the resource group where the resource exist. A new resource group will be created with the cluster, which will also be deleted with the cluster. Changing this forces a new resource to be created.
* `location` - (Required) The location where the cluster should be created. Changing this forces a new resource to be created.
* `master_profile` - (Required) A master profile block as documented below.
* `agent_pool_profiles` - (Required) One or more agent pool profile blocks as documented below.
* `linux_profile` - (Required) A Linux profile block as documented below.
* `windows_profile` - (Optional) A Windows profile block as documented below. This is required if any agent pools have `os_type` set to 'Windows'.
* `service_principal` - (Required) A service principal block as documented below.
* `kubernetes_version` - (Optional) The Kubernetes version running on the cluster.
* `tags` - (Optional) A mapping of tags to assign to the resource.

`master_profile` supports the following:

* `count` - (Required) Number of masters (VMs) in the container service cluster. Allowed values are 1, 3, and 5. The default value is 1.
* `dns_name_prefix` - (Required) The DNS prefix to use for the cluster master nodes.
* `vm_size` - (Optional) The VM size of each of the master VMs (e.g. Standard_F2 / Standard_D2v2). Changing this forces a new resource to be created.
* `osdisk_size` - (Optional) The master OS disk size in GB. Changing this forces a new resource.

`agent_pool_profile` supports the following:

* `name` - (Required) Unique name of the agent pool profile in the context of the subscription and resource group.
* `count` - (Required) Number of agents (VMs) to host containers. Allowed values must be in the rnge of 1 to 100 (inclusive). The default value is 1.
* `vm_size` - (Optional) The VM size of each of the agent pool VMs (e.g. Standard_F2 / Standard_D2v2). Changing this forces a new resource to be created.
* `os_disk_size` - (Optional) The agent OS disk size in GB. Changing this forces a new resource.
* `os_type` - (Optional) The Operating System used for the agent pools. Possible values are 'Linux' and Windows'. The default value is 'Linux'. 'Windows' is not officially supported. Changing this forces a new resource.

`linux_profile` supports the following:

* `admin_username` - (Required) The admin username for the cluster.
* `ssh` - (Required) An SSH key block as documented below.

`ssh` supports the following:

* `key_data` - (Required) The public SSH key used to access the cluster.

`windows_profile` supports the following:

* `admin_username` - (Required) The Windows admin username.
* `admin_password` - (Required) An Windows admin password.

`service_principal` supports the following:

* `client_id` - (Required) The ID for the service principal.
* `vault_id` - (Required) The Azure resource ID for the key vault containing the service principal secret.
* `secret_name` - (Required) The name of the key vault secret containing the value of your service principal secret.

## Attributes Reference

The following attributes are exported:

* `id` - The ACS Engine Kubernetes cluster resource ID
* `master_profile.0.fqdn` - FQDN for the master.
* `kube_config_raw` - Base64 encoded Kubernetes configuration.
* `kube_config` - Kubernetes configuration, sub-attributes defined below:
  * `host` - The Kubernetes cluster server host.
  * `username` - A username used to authenticate to the Kubernetes cluster.
  * `password` - A password or token used to authenticate to the Kubernetes cluster.
  * `client_certificate` - Base64 encoded public certificate used by clients to authenticate to the Kubernetes cluster.
  * `client_key` - Base64 encoded private key used by clients to authenticate to the Kubernetes cluster.
  * `cluster_ca_certificate` - Base64 encoded public CA certificate used as the root of trust for the Kubernetes cluster.
* `api_model` - Base64 encoded JSON model used for creating and updating the Kubernetes cluster.

## Import

ACS Engine clusters can be imported using the deployment resource ID and the directory containing their apimodel.json file delimited by a space. The file will need to be edited to work with the format expected by this provider. For details look at the `import documentation`.