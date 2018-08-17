#!/usr/bin/env bash

set -eo pipefail

TFDIR=scripts/keyvault

terraform_apply() {
    terraform init $TFDIR
    TF_VAR_tenant_id=$ARM_TENANT_ID TF_VAR_object_id=$ARM_OBJECT_ID TF_VAR_client_secret=$ARM_CLIENT_SECRET \
        terraform apply -input=false --auto-approve $TFDIR
}

terraform_destroy() {
    TF_VAR_tenant_id=$ARM_TENANT_ID TF_VAR_object_id=$ARM_OBJECT_ID TF_VAR_client_secret=$ARM_CLIENT_SECRET \
        terraform destroy -input=false --auto-approve $TFDIR
}

case "${1-}" in
    --tfapply)
    terraform_apply
    ;;
    --tfdestroy)
    terraform_destroy
    ;;
esac