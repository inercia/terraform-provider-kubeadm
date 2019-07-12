#!/bin/bash

###########################################################################################
# variables
###########################################################################################

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -f $DIR/common.bash  ] && source $DIR/common.bash
[ -f $DIR/dynamic.bash ] && source $DIR/dynamic.bash
[ -f $DIR/local.bash   ] && source $DIR/local.bash

NUM_MASTERS=3
NUM_WORKERS=2

cd $E2E_ENV

TF_ARGS=""
[ -f "ci.tfvars" ] && [ "$CI" = "true" ] && TF_ARGS="$TF_ARGS -var-file=ci.tfvars"

export KUBECONFIG=$E2E_ENV/kubeconfig.local

###########################################################################################
# increase one master
###########################################################################################

# hack: Terraform-docker fails to restart the haproxy, so just kill it or the "apply" will fail
#       we need to do this before changing the number of masters
docker_stop "kubeadm-haproxy"

log "Adding one master..."
tf_apply $NUM_MASTERS $NUM_WORKERS $TF_ARGS

###########################################################################################
# checks
###########################################################################################

check_exp_nodes $NUM_MASTERS $NUM_WORKERS
