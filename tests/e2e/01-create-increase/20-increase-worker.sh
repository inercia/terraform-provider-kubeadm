#!/bin/bash

###########################################################################################
# variables
###########################################################################################

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -f $DIR/common.bash  ] && source $DIR/common.bash
[ -f $DIR/dynamic.bash ] && source $DIR/dynamic.bash
[ -f $DIR/local.bash   ] && source $DIR/local.bash

NUM_MASTERS=3
NUM_WORKERS=3

cd $E2E_ENV

TF_ARGS=""
[ -f "ci.tfvars" ] && [ "$CI" = "true" ] && TF_ARGS="$TF_ARGS -var-file=ci.tfvars"

###########################################################################################
# increase one worker
###########################################################################################

echo ">>> Adding one worker..."
TF_VAR_master_count=$NUM_MASTERS TF_VAR_worker_count=$NUM_WORKERS \
    terraform apply -auto-approve $TF_ARGS
if [ $? -ne 0 ] ; then
    echo ">>> FAILED: could not add one worker"
    exit 1
fi

###########################################################################################
# checks
###########################################################################################

echo ">>> Checking we can get cluster info with kubectl..."
kubectl --kubeconfig=kubeconfig.local get nodes
if [ $? -ne 0 ] ; then
    echo ">>> FAILED: could not get the nodes with kubectl"
    exit 1
fi

OUT=$(kubectl --kubeconfig=kubeconfig.local get nodes --show-labels)
if [ $? -ne 0 ] ; then
    echo ">>> FAILED: could not get the number of nodes with kubectl"
    exit 1
fi

EXP_NUM_NODES=$((NUM_MASTERS + NUM_WORKERS))
echo ">>> Checking we have $EXP_NUM_NODES nodes..."
CURR_NUM_NODES=$(echo "$OUT" | grep -c "kubernetes.io/hostname")
if [ $CURR_NUM_NODES -ne $EXP_NUM_NODES ] ; then
    echo ">>> FAILED: current number of nodes, $CURR_NUM_NODES, do not match $EXP_NUM_NODES"
    exit 1
fi

EXP_NUM_MASTERS=$NUM_MASTERS
echo ">>> Checking we have $EXP_NUM_MASTERS masters..."
CURR_NUM_MASTERS=$(echo "$OUT" | grep -c "node-role.kubernetes.io/master")
if [ $CURR_NUM_MASTERS -ne $EXP_NUM_MASTERS ] ; then
    echo ">>> FAILED: current number of masters, $CURR_NUM_MASTERS, do not match $EXP_NUM_MASTERS"
    exit 1
fi

