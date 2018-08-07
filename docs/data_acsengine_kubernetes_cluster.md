# Data Source: acsengine_kubernetes_cluster

The ACS Engine data source allows access to details of a specific ACS Engine cluster.

## Example Usage

If you had a cluster with Terraform address `acsengine_kubernetes_cluster.cluster`:

```hcl
data "acsengine_kubernetes_cluster" "cluster" {
    name = "${acsengine_kubernetes_cluster.cluster.name}"
    resource_group  = "${acsengine_kubernetes_cluster.cluster.resource_group}"
    api_model = "${acsengine_kubernetes_cluster.cluster.api_model}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ACS Engine cluster.
* `resource_group` - (Required) The name of the resource group in which the ACS Engine cluster exists.
* `api_model` - (Required) Base64 encoded JSON model used for creating and updating the Kubernetes cluster.

## Attributes Reference

The following attributes are supported:

* `id` - The ACS Engine Kubernetes cluster resource ID
* `kube_config_raw` - Base64 encoded Kubernetes configuration.
* `kube_config` - A `kube_config` block as defined below.
* `location` - The Azure region in which the ACS Engine cluster exists.
* `linux_profile` - A `linux_profile` block as defined below.
* `service_principal`- A `service_principal` block as defined below.
* `master_profile` - A `master_profile` block as defined below.
* `agent_pool_profiles` - A `agent_pool_profiles` block as defined below.
* `tags` - A mapping of tags assigned to the resource group created to contain this resource.

`kube_config` exports the following:

* `host` - The Kubernetes cluster server host.
* `username` - A username used to authenticate to the Kubernetes cluster.
* `password` - A password or token used to authenticate to the Kubernetes cluster.
* `client_certificate` - Base64 encoded public certificate used by clients to authenticate to the Kubernetes cluster.
* `client_key` - Base64 encoded private key used by clients to authenticate to the Kubernetes cluster.
* `cluster_ca_certificate` - Base64 encoded public CA certificate used as the root of trust for the Kubernetes cluster.

`linux_profile` exports the following:

* `admin_username` - The admin username for the cluster.
* `ssh` - An SSH key block as documented below.

`ssh` exports the following:

* `key_data` - The public SSH key used to access the cluster.

`service_principal` exports the following:

* `client_id` - The ID for the service principal.

`master_profile` supports the following:

* `count` - Number of masters (VMs) in the container service cluster.
* `dns_name_prefix` - The DNS prefix to use for the cluster master nodes.
* `vm_size` - The VM size of each of the master VMs (e.g. Standard_F2 / Standard_D2v2).
* `osdisk_size` - The master OS disk size in GB. Changing this forces a new resource.

`agent_pool_profile` supports the following:

* `name` - Unique name of the agent pool profile in the context of the subscription and resource group.
* `count` - Number of agents (VMs) to host containers.
* `vm_size` - The VM size of each of the agent pool VMs (e.g. Standard_F2 / Standard_D2v2).
* `os_disk_size` - The agent OS disk size in GB. Changing this forces a new resource.
* `os_type` - The Operating System used for the agent pools.