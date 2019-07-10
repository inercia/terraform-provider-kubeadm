#!/usr/bin/env bash

export PATH=/opt/bin:$PATH

log()      { echo >&2 ">>> $@"; }
failed()   { log "FAILED: $@" ; }
warn()     { log "WARNING: $@" ; }
abort()    { log "FATAL: $@" ; exit 1 ; }

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

