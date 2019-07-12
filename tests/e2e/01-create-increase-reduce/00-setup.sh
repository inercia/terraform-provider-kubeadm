#!/bin/bash

###########################################################################################
# variables
###########################################################################################

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -f $DIR/common.bash  ] && source $DIR/common.bash
[ -f $DIR/dynamic.bash ] && source $DIR/dynamic.bash
[ -f $DIR/local.bash   ] && source $DIR/local.bash

TF_ARGS=""
NUM_MASTERS=2
NUM_WORKERS=2

cd $E2E_ENV
[ -f "ci.tfvars" ] && [ "$CI" = "true" ] && TF_ARGS="$TF_ARGS -var-file=ci.tfvars"

export KUBECONFIG=$E2E_ENV/kubeconfig.local

###########################################################################################
# cluster creation
###########################################################################################

section "Terraform info"
terraform --version

section "Docker info"
docker info

section "Initializing test env"
terraform init
[ $? -eq 0 ] || abort "could not init Terraform"

section "Creating initial cluster..."
tf_apply $NUM_MASTERS $NUM_WORKERS $TF_ARGS

###########################################################################################
# checks
###########################################################################################

check_exp_nodes $NUM_MASTERS $NUM_WORKERS

