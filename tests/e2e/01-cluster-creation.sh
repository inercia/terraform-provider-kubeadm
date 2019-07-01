#!/bin/bash

[ -f common.bash  ] && source common.bash
[ -f dynamic.bash ] && source dynamic.bash
[ -f local.bash   ] && source local.bash

TF_ARGS=""

cd $E2E_ENV

[ -f "ci.tfvars" ] && TF_ARGS="$TF_ARGS -var-file=ci.tfvars"

echo ">>> Terraform info:"
terraform --version

echo ">>> Docker info:"
docker info

echo ">>> Initializing test env..."
terraform init
if [ $? -ne 0 ] ; then
    echo ">>> FAILED: could not init"
    exit 1
fi

echo ">>> Creating cluster..."
terraform apply -auto-approve $TF_ARGS
if [ $? -ne 0 ] ; then
    echo ">>> FAILED: could not create cluster"
    exit 1
fi

echo ">>> Getting cluster info with kubectl..."
kubectl --kubeconfig=kubeconfig.local get nodes --show-labels
if [ $? -ne 0 ] ; then
    echo ">>> FAILED: could not get the number of nodes with kubectl"
    exit 1
fi

