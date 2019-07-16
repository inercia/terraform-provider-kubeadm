#!/usr/bin/env bash

###########################################################################################
# variables
###########################################################################################

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -f $DIR/common.bash  ] && source $DIR/common.bash
[ -f $DIR/dynamic.bash ] && source $DIR/dynamic.bash
[ -f $DIR/local.bash   ] && source $DIR/local.bash

[ -d $E2E_ENV ] || abort "directory $E2E_ENV does not seem to exist"
cd $E2E_ENV

TF_ARGS=""
[ -f "ci.tfvars" ] && [ "$IS_CI" = "true" ] && TF_ARGS="$TF_ARGS -var-file=ci.tfvars"

export KUBECONFIG=$E2E_ENV/kubeconfig.local

###########################################################################################
# list all the tokens
###########################################################################################

section "Showing all the tokens..."
kubeadm_token_list
