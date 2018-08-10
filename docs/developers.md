# Developer Guide

## Prerequisites

* Go 1.10.0 or later
* kubectl 1.10 or later
* An Azure account (to deploy VMs and Azure infrastructure)
* Git

## Structure of Code

The code for this project is organized as follows:

* The `acsengine` folder contains most of the Go code specific to configuring acs-engine Kubernetes clusters through Terraform. This folder also contains tests for files in that folder.
* The `docs` folder contains user and developer documentation and examples
* The `vendor` folder is managed by Golang Dep and should not be modified except through [Gopkg.toml](https://github.com/shanalily/terraform-provider-acsengine/blob/master/Gopkg.toml).

## Git Conventions

We use Git for our version control system. The `master` branch is the home of current development. Releases will eventually be tagged...

We accept changes to code via GitHub pull requests. Look at [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on submitting a PR.

## Third Party Dependencies

Third party dependencies reside locally inside the respository under the `vendor` directory. The dependency manager used is [dep](https://golang.github.io/dep/). Changes to dependencies can be made at [Gopkg.toml](https://github.com/shanalily/terraform-provider-acsengine/blob/master/Gopkg.toml).

If there seem to be dependencies missing, you can run `make vendor`. If you want to check which versions are being used, you can run `make vendor-status`.

## Go Conventions

We follow the Go coding style standards.

To check that your code meets our standards, run `make lint`. This will run `gometalinter`.

## Unit Tests

Unit tests can run via `make test`. This will not create any Azure resources.

## End-to-End Tests

There are Terraform acceptance tests which create, test, and destroy a cluster. These can be run with `make testacc`. These will create actual Azure resources. There may be cases where a test stops prematurely and resources are not deleted, so you may want to make sure you do not have unused VMs and resource groups leftover if tests fail.

You will need to have the following environment variables set:

* `ARM_CLIENT_ID`: Azure client ID
* `ARM_CLIENT_SECRET`: Azure client secret
* `ARM_SUBSCRIPTION_ID`: Azure subscription UUID
* `ARM_TENANT_ID`: Azure tenant UUID
* `SSH_KEY_PUB`: Public SSH key

<!-- ## Debugging

Delve can be used for debugging... more on this later. -->