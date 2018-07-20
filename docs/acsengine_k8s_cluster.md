# azurerm_acsengine_k8s_cluster

Manages AKS Cluster

**Note:** All arguments including the client secret will be stored in the raw state as plain-text.

## Example Usage

<!-- Try testing this exact configuration -->
```hcl
resource "acsengine_k8s_cluster" "test" {
	name               = "acctest"
	resource_group     = "acctestRG"
	location           = "southcentralus"
	kubernetes_version = "1.10.4"

	master_profile {
		count           = 1
		dns_name_prefix = "acctestmaster"
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
		admin_username = "acctestuser"
		ssh {
			key_data = "ssh-rsa AAAAB3NzaC... terraform@demo.tld"
		}
	}

	service_principal {
		client_id     = ""
		client_secret = ""
	}

	tags {
		Environment = "Production"
	}
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the cluster to create. Changing this forces a new resource to be created.
* `resource_group` - (Required) Specifies the name of the resource group where the resource exist. Changing this forces a new resource to be created.
* `location` - (Required) The location where the cluster should be created. Changing this forces a new resource to be created.
* `master_profile` - (Required) A master profile block as documented below.
* `agent_pool_profiles` - (Required) One or more agent pool profile blocks as documented below.
* `linux_profile` - (Required) A Linux profile block as documented below.
* `service_principal` - (Required) A service principal block as documented below.
* `kubernetes_version` - (Optional) The Kubernetes version running on the cluster.
* `tags` - (Optional) A mapping of tags to assign to the resource.

`master_profile` supports the following:

* `count` - (Required) Number of masters (VMs) in the container service cluster. Allowed values are 1, 3, and 5. The default value is 1.
* `dns_name_prefix` - (Required) The DNS prefix to use for the cluster master nodes.
* `vm_size` - (Optional) The VM size of each of the master VMs (e.g. Standard_F2 / Standard_D2v2). Changing this forces a new resource to be created.

<!-- * `osdisk_size` - (Optional) Size in GB of the OS disk for each master node. -->

`agent_pool_profile` supports the following:

* `name` - (Required) Unique name of the agent pool profile in the context of the subscription and resource group.
* `count` - (Required) Number of agents (VMs) to host containers. Allowed values must be in the rnge of 1 to 100 (inclusive). The default value is 1.
* `vm_size` - (Optional) The VM size of each of the agent pool VMs (e.g. Standard_F2 / Standard_D2v2). Changing this forces a new resource to be created.
* `os_disk_size` - (Optional) The agent operation system disk size in GB. Changing this forces a new resource.
* `os_type` - (Optional) The Operating System used for the agent pools. Possible values are 'Linux' and Windows'. The default value is 'Linux' and 'Windows' is not officially supported. Changing this forces a new resource.

<!-- * `os_type` - (Optional) The OS type of each of the agent pool VMs. Allowed values are Linux and Windows. The default value is Linux. Changing this forces a new resource to be created. -->
<!-- * `osdisk_size` - (Optional) Size in GB of the OS disk for each of the agent pool VMs. -->

`linux_profile` supports the following:

* `admin_username` - (Required) The admin username for the cluster.
* `ssh` - (Required) An SSH key block as documented below.

`ssh` supports the following:

* `key_data` - (Required) The public SSH key used to access the cluster.

`service_principal` supports the following:

* `client_id` - (Required) The ID for the service principal.
* `client_secret` - (Required) The secret password associated with the service principal.

## Attributes Reference

The following attributes are exported:

* `id` - The ACS Engine Kubernetes cluster resource ID
* `master_profile.fqdn` - FQDN for the master.
* `kube_config_raw` - Base64 encoded Kubernetes configuration.
* `kube_config` - Kubernetes configuration, sub-attributes defined below:
	* `host` - The Kubernetes cluster server host.
	* `username` - A username used to authenticate to the Kubernetes cluster.
	* `password` - A password or token used to authenticate to the Kubernetes cluster.
	* `client_certificate` - Base64 encoded public certificate used by clients to authenticate to the Kubernetes cluster.
	* `client_key` - Base64 encoded private key used by clients to authenticate to the Kubernetes cluster.
	* `cluster_ca_certificate` - Base64 encoded public CA certificate used as the root of trust for the Kubernetes cluster.
