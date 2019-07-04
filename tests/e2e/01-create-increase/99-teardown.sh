#!/bin/bash

###########################################################################################
# variables
###########################################################################################

[ -f common.bash  ] && source common.bash
[ -f dynamic.bash ] && source dynamic.bash
[ -f local.bash   ] && source local.bash

TF_ARGS=""

cd $E2E_ENV
[ -f "ci.tfvars" ] && [ "$CI" = "true" ] && TF_ARGS="$TF_ARGS -var-file=ci.tfvars"


###########################################################################################
# cluster creation
###########################################################################################

terraform destroy --auto-approve $TF_ARGS

