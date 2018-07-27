# Notes on Upgrading Kubernetes Version on Cluster

Like ACS Engine, this provider allows you to upgrade the Kubernetes version running on your cluster. However, there are restrictions about what versions you can upgrade to from your current version. You can only upgrade one minor version at a time. Those restrictions are outlined in this [ACS Engine doc](https://github.com/Azure/acs-engine/tree/master/examples/k8s-upgrade).

Basically, if you have acs-engine installed, you can run
```bash
acs-engine orchestrators --orchestrator Kubernetes --version 1.8.13
```

where 1.8.13 can be replaced by the version you are currently on. The listed versions are allowed for upgrading. This Terraform provider will also quickly give you an error if you give an invalid version and run `terraform plan` or `terraform apply` (I think).