#!/usr/bin/env bash

###########################################################################################
# variables
###########################################################################################

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -f $DIR/common.bash  ] && source $DIR/common.bash
[ -f $DIR/dynamic.bash ] && source $DIR/dynamic.bash
[ -f $DIR/local.bash   ] && source $DIR/local.bash

cd $E2E_ENV

TF_ARGS=""
[ -f "ci.tfvars" ] && [ "$CI" = "true" ] && TF_ARGS="$TF_ARGS -var-file=ci.tfvars"

KUBECONFIG=$E2E_ENV/kubeconfig.local

###########################################################################################
# list all the tokens
###########################################################################################

[ -f $KUBECONFIG ] || abort "no kubeconfig found at $KUBECONFIG"

command -v kubeadm >/dev/null 2>&1 || { log "kubeadm is not installed: installing." ; install_kubeadm ; }

log "Current list of tokens:"
kubeadm token list --kubeconfig=$KUBECONFIG

