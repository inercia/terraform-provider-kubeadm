#!/bin/bash

###########################################################################################
# variables
###########################################################################################

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -f $DIR/common.bash  ] && source $DIR/common.bash
[ -f $DIR/dynamic.bash ] && source $DIR/dynamic.bash
[ -f $DIR/local.bash   ] && source $DIR/local.bash

NUM_MASTERS=2
NUM_WORKERS=2

[ -d $E2E_ENV ] || abort "directory $E2E_ENV does not seem to exist"
cd $E2E_ENV

TF_ARGS=""
[ -f "ci.tfvars" ] && [ "$IS_CI" = "true" ] && TF_ARGS="$TF_ARGS -var-file=ci.tfvars"

export KUBECONFIG=$E2E_ENV/kubeconfig.local

###########################################################################################
# increase one worker
###########################################################################################

section "Deleting a worker..."
tf_apply $NUM_MASTERS $NUM_WORKERS $TF_ARGS

###########################################################################################
# checks
###########################################################################################

check_exp_nodes $NUM_MASTERS $NUM_WORKERS
