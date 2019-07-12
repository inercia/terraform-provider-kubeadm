#!/usr/bin/env bash

# common Terraform arguments
export COMMON_TF_ARGS="--auto-approve"

export PATH=/opt/bin:$PATH

######################################################################################

log()      { echo >&2 ">>> $@"; }
failed()   { log "FAILED: $@" ; }
warn()     { log "WARNING: $@" ; }
abort()    { log ">>>>>>>>>> FATAL: $@ <<<<<<<<<<< <<<" ; exit 1 ; }
section()  {
    log " ---------------------------------------------------------------"
    log $@
    log " ---------------------------------------------------------------"
}

######################################################################################

download_k8s_bin() {
    bin=$1

    RELEASE="$(curl -sSL https://dl.k8s.io/release/stable.txt)"

    log "Downloading $bin..."
    cd /tmp
    curl -L --remote-name-all https://storage.googleapis.com/kubernetes-release/release/${RELEASE}/bin/linux/amd64/$bin
    [ -f $bin ] || abort "$bin was not downloaded with curl"

    log "Moving $bin to /opt/bin with the right permissions ..."
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

    echo ">>> Checking we can get cluster info with kubectl..."
    kubectl --kubeconfig=$KUBECONFIG get nodes
    [ $? -eq 0 ] || abort "could not get the nodes with kubectl"

    local output=$(kubectl --kubeconfig=$KUBECONFIG get nodes --show-labels)
    [ $? -eq 0 ] || abort "could not get the number of nodes with kubectl"

    exp_num_nodes=$((exp_num_masters + exp_num_workers))
    echo ">>> Checking we have $exp_num_nodes nodes..."
    curr_num_nodes=$(echo "$output" | grep -c "kubernetes.io/hostname")
    if [ $curr_num_nodes -ne $exp_num_nodes ] ; then
        abort "current number of nodes, $curr_num_nodes, do not match $exp_num_nodes"
    fi

    echo ">>> Checking we have $exp_num_masters masters..."
    curr_num_masters=$(echo "$output" | grep -c "node-role.kubernetes.io/master")
    if [ $curr_num_masters -ne $exp_num_masters ] ; then
        abort "current number of masters, $curr_num_masters, do not match $exp_num_masters"
    fi
}

######################################################################################

tf_apply() {
    local num_masters=$1
    shift
    local num_workers=$1
    shift

    TF_VAR_master_count=$num_masters TF_VAR_worker_count=$num_workers \
        terraform apply $COMMON_TF_ARGS $@
    [ $? -eq 0 ] || \
        abort "could not apply Terraform script for masters:$num_masters workers:$num_workers"
}

tf_destroy() {
    terraform destroy $COMMON_TF_ARGS $@
    [ $? -eq 0 ] || \
        abort "could not destroy Terraform cluster"
}

######################################################################################

kubeadm_check_installation() {
    command -v kubeadm >/dev/null 2>&1 || { log "kubeadm is not installed: installing." ; install_kubeadm ; }
}

kubeadm_token_list() {
    kubeadm_check_installation

    [ -f $KUBECONFIG ] || abort "no kubeconfig found at $KUBECONFIG"

    log "Current list of tokens:"
    kubeadm token list --kubeconfig=$KUBECONFIG
}

kubeadm_token_flush() {
    kubeadm_check_installation

    [ -f $KUBECONFIG ] || abort "no kubeconfig found at $KUBECONFIG"

    TOKENS=$(kubeadm token list --kubeconfig=$KUBECONFIG | grep -E "[a-z0-9]{6}\.[a-z0-9]{16}" | cut -f1 -d" ")
    log "Removing all the tokens..."
    for token in $TOKENS ; do
        log "... removing token $token"
        kubeadm token delete --kubeconfig=$KUBECONFIG $token
    done

    log "Listing tokens after 'flush':"
    kubeadm_token_list
}

######################################################################################

docker_stop() {
    local name=$1

    docker stop $name
}
