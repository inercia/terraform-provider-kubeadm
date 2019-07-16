#!/usr/bin/env bash

export PATH=/opt/bin:$PATH

# common Terraform arguments
export TF_COMMON_ARGS="--auto-approve"

export TF_LOG_FILENAME="$E2E_ENV/terraform.log"


######################################################################################

RED="\e[31m"
GREEN="\e[32m"
YELLOW="\e[33m"
NC="\e[0m"
BOLD="\e[1m"

log()      { echo -e >&2 ">>> $@"; }
info()     { log "${INFO} $@${NC}" ;}
failed()   { log "${RED}FAILED: $@${NC}" ; }
warn()     { log "${RED}WARNING: $@${NC}" ; }
abort()    { log "${RED}${BOLD}>>>>>>>>>> FATAL: $@ <<<<<<<<<<< <<<${NC}" ; exit 1 ; }
section()  {
    log "${GREEN} ---------------------------------------------------------------${NC}"
    log "${GREEN} $@${NC}"
    log "${GREEN} ---------------------------------------------------------------${NC}"
}

######################################################################################

download_k8s_bin() {
    bin=$1

    RELEASE="$(curl -sSL https://dl.k8s.io/release/stable.txt)"

    info "Downloading $bin..."
    cd /tmp
    curl -L --remote-name-all https://storage.googleapis.com/kubernetes-release/release/${RELEASE}/bin/linux/amd64/$bin
    [ -f $bin ] || abort "$bin was not downloaded with curl"

    info "Moving $bin to /opt/bin with the right permissions ..."
    chmod 755 $bin
    mkdir -p /opt/bin
    sudo mv $bin /opt/bin/
    [ -x /opt/bin/$bin ] || abort "$bin was not properly installed in /opt/bin"
}

install_kubeadm() {
    download_k8s_bin kubeadm
}

install_kubectl() {
    download_k8s_bin kubectl
}

check_exp_nodes() {
    local exp_num_masters=$1
    local exp_num_workers=$2

    [ -f $KUBECONFIG ] || abort "no kubeconfig found at $KUBECONFIG"

    info "Checking we can get cluster info with kubectl..."
    kubectl --kubeconfig=$KUBECONFIG get nodes
    [ $? -eq 0 ] || abort "could not get the nodes with kubectl"

    local output=$(kubectl --kubeconfig=$KUBECONFIG get nodes --show-labels)
    [ $? -eq 0 ] || abort "could not get the number of nodes with kubectl"

    exp_num_nodes=$((exp_num_masters + exp_num_workers))
    info "Checking we have $exp_num_nodes nodes..."
    curr_num_nodes=$(echo "$output" | grep -c "kubernetes.io/hostname")
    if [ $curr_num_nodes -ne $exp_num_nodes ] ; then
        abort "current number of nodes, $curr_num_nodes, do not match $exp_num_nodes"
    fi
    info "... good, we have $exp_num_nodes nodes..."

    info "Checking we have $exp_num_masters masters..."
    curr_num_masters=$(echo "$output" | grep -c "node-role.kubernetes.io/master")
    if [ $curr_num_masters -ne $exp_num_masters ] ; then
        abort "current number of masters, $curr_num_masters, do not match $exp_num_masters"
    fi
    info "... good, we have $exp_num_masters masters..."
}

######################################################################################

tf_apply() {
    local num_masters=$1
    shift
    local num_workers=$1
    shift

    local log_filename="$E2E_ENV/terraform.log"

    info "running 'terraform apply' for masters:$num_masters workers:$num_workers"
    exec 3>$TF_LOG_FILENAME
    TF_LOG=DEBUG \
    TF_VAR_master_count=$num_masters TF_VAR_worker_count=$num_workers \
        terraform apply $TF_COMMON_ARGS $@ 2>&3 | tee -a >(tee >&3)
    res=$?
    exec 3>&-

    [ $res -eq 0 ] || \
        abort "could not apply Terraform script for masters:$num_masters workers:$num_workers"
    info "'terraform apply' was successful"
}

tf_destroy() {
    local log_filename="$E2E_ENV/terraform.log"

    info "running 'terraform destroy'"
    exec 3>$TF_LOG_FILENAME
    TF_LOG=DEBUG \
        terraform destroy $TF_COMMON_ARGS $@ 2>&3 | tee -a >(tee >&3)
    res=$?
    exec 3>&-

    [ $res -eq 0 ] || abort "could not destroy Terraform cluster"
    info "'terraform destroy' was successful"
}

######################################################################################

kubeadm_check_installation() {
    command -v kubeadm >/dev/null 2>&1 || { log "kubeadm is not installed: installing." ; install_kubeadm ; }
}

kubeadm_token_list() {
    kubeadm_check_installation

    [ -f $KUBECONFIG ] || abort "no kubeconfig found at $KUBECONFIG"

    info "Current list of tokens:"
    kubeadm token list --kubeconfig="$KUBECONFIG"
}

kubeadm_token_flush() {
    kubeadm_check_installation

    [ -f $KUBECONFIG ] || abort "no kubeconfig found at $KUBECONFIG"

    TOKENS=$(kubeadm token list --kubeconfig="$KUBECONFIG" | grep -E "[a-z0-9]{6}\.[a-z0-9]{16}" | cut -f1 -d" ")
    info "Removing all the tokens..."
    for token in $TOKENS ; do
        info "... removing token $token"
        kubeadm token delete --kubeconfig="$KUBECONFIG" $token
    done

    info "Listing tokens after 'flush':"
    kubeadm_token_list
}

######################################################################################

docker_stop() {
    local name=$1

    docker stop $name
}
