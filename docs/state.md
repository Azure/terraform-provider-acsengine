# Terraform State of ACS Engine Cluster

Perhaps you are wondering where the state of the cluster is being read from. If you are familiar with ACS Engine, you know that an `apimodel.json` file is generated when you run `acs-engine generate`. The contents of this file are stored in the `terraform.tfstate` file as the value of `api_model`. This is where the state is read from, and it is updated as needed. This is done with the assumption that the cluster will only be changed through terraform configurations.

Unfortunately, this makes implementing the acs-engine kubernetes cluster data source as well as importing less straightforward. These are currently not supported.

## Note on resources created

Storing the contents of `apimodel.json` in the Terraform state means that no new resources have to be created to store this information. The Azure resources created include the Azure resource group for the cluster, and all resources that are essential to creating and deploying a cluster (for instance, VMs for nodes and agent pools). The resource group is deleted to destroy the cluster. **Important:** This means that new resources should not be created within this resource group unless they can be deleted with the cluster.