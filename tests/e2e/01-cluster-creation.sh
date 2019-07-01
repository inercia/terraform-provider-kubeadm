[ -f config.bash  ] && source config.bash
[ -f dynamic.bash ] && source dynamic.bash
[ -f local.bash   ] && source local.bash

cd $E2E_ENV

terraform --version

terraform init

terraform apply -auto-approve

terraform output

terraform destroy



