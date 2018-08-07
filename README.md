# ACS Engine Kubernetes Terraform Provider
[![CircleCI](https://circleci.com/gh/Azure/terraform-provider-acsengine/tree/master.svg?style=svg)](https://github.com/Azure/terraform-provider-acsengine)

## Overview

The Azure Container Service Engine Kubernetes Terraform Provider allows you to create and manage [ACS Engine](https://github.com/Azure/acs-engine) Kubernetes clusters with a simple Terraform configuration. Other container orchestrators are not supported.

Note: This is very much still a work in progress (by an intern) :)

This started out as a fork of [terraform-providers/terraform-provider-azurerm](https://github.com/terraform-providers/terraform-provider-azurerm/tree/master/azurerm) so a lot of code is inspired-by-slash-taken-from that repo.

## User Guides

* [Usage](docs/acsengine_k8s_cluster.md) - details about Kubernetes resource schema and how to configure a cluster
* [Scaling clusters](docs/scaling-agent-pools.md) - shows how to scale a cluster's agent pool count
* [Upgrading clusters](docs/upgrading-clusters.md) - shows how to upgrade a cluster's Kubernetes version
* [Terraform state](docs/state.md) - notes on how the state of the cluster is stored and resource creation
* [Developer guide](docs/developers.md)

## General Requirements

* [Terraform](https://www.terraform.io/downloads.html) 0.11.x
* [Go](https://golang.org/doc/install) 1.10.x (to build the provider plugin)

## Building The Provider

Clone repository to: `$GOPATH/src/github.com/Azure/terraform-provider-acsengine`

```sh
$ mkdir -p $GOPATH/src/github.com/Azure; cd $GOPATH/src/github.com/Azure
$ git clone git@github.com:Azure/terraform-provider-acsengine
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/Azure/terraform-provider-acsengine
$ make build
```

## Using the provider

```
# Configure the Microsoft ACS Engine Provider
provider "acsengine" {
  # NOTE: Environment Variables can also be used for Service Principal authentication
  # Terraform also supports authenticating via the Azure CLI too.
  # see here for more info: http://terraform.io/docs/providers/azurerm/index.html

  # subscription_id = "..."
  # client_id       = "..."
  # client_secret   = "..."
  # tenant_id       = "..."
}

# Create a Kubernetes cluster
resource "acsengine_kubernetes_cluster" "test" {
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

Further usage documentation can be found in the `docs` directory.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.9+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-acsengine
...
```

In order to run the provider unit tests, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`. This will take some time.

The following ENV variables must be set in your shell prior to running acceptance tests:

* ARM_CLIENT_ID
* ARM_CLIENT_SECRET
* ARM_SUBSCRIPTION_ID
* ARM_TENANT_ID
* ARM_TEST_LOCATION
* SSH_KEY_PUB

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```

## Contributing

This project welcomes contributions and suggestions. Please follow the guidelines on our [contributing page](CONTRIBUTING.md) if you would like to help out.

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

## Code of Conduct

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
