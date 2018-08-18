#!/usr/bin/env bash

set -eo pipefail

TFDIR=scripts/keyvault

terraform_apply() {
    terraform init $TFDIR
    TF_VAR_tenant_id=$ARM_TENANT_ID TF_VAR_sp_object_id=$ARM_SP_OBJECT_ID TF_VAR_user_object_id=$ARM_USER_OBJECT_ID TF_VAR_client_secret=$ARM_CLIENT_SECRET \
        terraform apply -input=false --auto-approve $TFDIR
}

terraform_destroy() {
    TF_VAR_tenant_id=$ARM_TENANT_ID TF_VAR_sp_object_id=$ARM_SP_OBJECT_ID TF_VAR_user_object_id=$ARM_USER_OBJECT_ID TF_VAR_client_secret=$ARM_CLIENT_SECRET \
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