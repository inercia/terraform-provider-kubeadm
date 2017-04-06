#!/bin/sh

##########################################################################################
# kubeadm setup script
##########################################################################################

LSB_RELEASE=/usr/bin/lsb_release

# we will try to discover the DIST and RELEASE
ID=
DIST=
RELEASE=

# a runner script we will use for running kubeadm
# in order to use it with Terraform, we must run the command in the background
KUBEADM_RUNNER=/tmp/kubeadm

# a container that can be used for running kubeadm
DOCKER_RUNNER="inercia/kubeadm"

##########################################################################################

log()    { echo ">>> kubeadm-setup: $@" ;     }
abort()  { log "$@" ; exit 1 ; }

##########################################################################################

# installation for SUSE variants: OpenSUSE/SLE/CaaSP...
install_zypper() {
    zypper_args="-n --no-gpg-checks --quiet --no-color"

    source /etc/os-release
    case $NAME in
      "CAASP")
        containers_repo_url="http://download.opensuse.org/repositories/Virtualization:/containers/SLE_12_SP1/"
        ;;
      *)
        containers_repo_url="http://download.opensuse.org/repositories/Virtualization:/containers/$(echo -n $PRETTY_NAME | tr " " "_")"
        ;;
    esac

	log "installing for SUSE"
	if [ ! -f /etc/zypp/repos.d/containers.repo ] ; then
        zypper $zypper_args ar -Gf "$containers_repo_url" containers || abort "could not enable containers repo"
		zypper $zypper_args ref containers                           || abort "could not refresh the containers repo"
	else
		log "repository already found: skipping installation of the repo"
	fi

    log "checking we have everything we need..."
	[ -x /usr/bin/kubelet ] || zypper $zypper_args in -y kubernetes-kubelet || abort "could not finish the installation of kubelet"
	[ -x /usr/bin/docker  ] || zypper $zypper_args in -y docker             || abort "could not finish the installation of docker"
	[ -x /usr/bin/kubeadm ] || zypper $zypper_args in -y kubernetes-kubeadm || abort "could not finish the installation of kubeadm"
	[ -x /usr/bin/kubeclt ] || zypper $zypper_args in -y kubernetes-client  || abort "could not finish the installation of kubectl"
    log "... everything installed"

    log "starting services"
    systemctl enable docker  && systemctl start docker  || abort "could not start docker"
    systemctl enable kubelet && systemctl start kubelet || abort "could not start the kubelet"

	install_kubeadm_runner /usr/bin/kubeadm
}

# installation for RedHat variants: RedHat/CentOS...
install_yum() {
	log "Installing for RedHat..."

	[ -n "$RELEASE" ] || abort "can't continue without knowing the release"

	if [ ! -f /etc/yum.repos.d/kubernetes.repo ] ; then
		cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=http://yum.kubernetes.io/repos/kubernetes-el$RELEASE-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
		setenforce 0

	else
		log "repository already found: skipping installation of the repo"
	fi

    log "checking we have everything we need..."
    [ -x /usr/bin/kubelet ] || yum install -y kubelet kubernetes-cni || abort "could not finish the installation of kubelet"
    [ -x /usr/bin/docker  ] || yum install -y docker                 || abort "could not finish the installation of docker"
    [ -x /usr/bin/kubeadm ] || yum install -y kubeadm                || abort "could not finish the installation of kubeadm"
    [ -x /usr/bin/kubectl ] || yum install -y kubectl                || abort "could not finish the installation of kubectl"
    log "... everything installed"

    log "starting services"
    systemctl enable docker  && systemctl start docker  || abort "could not start docker"
    systemctl enable kubelet && systemctl start kubelet || abort "could not start the kubelet"

	install_kubeadm_runner /usr/bin/kubeadm
}

# installation for Debian variants: debian/Ubuntu...
install_apt() {
	log "installing for Ubuntu|Debian..."
	apt-get update && apt-get install -y apt-transport-https
	if [ ! -f /etc/apt/sources.list.d/kubernetes.list ] ; then
		curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -

		cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF
		apt-get update
	else
		log "repository already found: skipping installation of the repo"
	fi

    log "checking we have everything we need..."
    [ -x /usr/bin/kubelet ] || apt-get install -y kubelet kubernetes-cni || abort "could not finish the installation of kubelet"
    [ -x /usr/bin/docker  ] || apt-get install -y docker.io              || abort "could not finish the installation of docker"
    [ -x /usr/bin/kubeadm ] || apt-get install -y kubeadm                || abort "could not finish the installation of kubeadm"
    [ -x /usr/bin/kubectl ] || apt-get install -y kubectl                || abort "could not finish the installation of kubectl"
    log "... everything installed"

	install_kubeadm_runner /usr/bin/kubeadm
}

# install a script for running kubeadm inside a container
# TODO: this is not functional yet!!!
install_docker_runner() {
    log "installing docker runnner for kubeadm"
	cat <<EOF >$KUBEADM_RUNNER
#!/bin/sh
docker run --rm -ti --privileged --net=host \
        -v /etc/kubernetes:/etc/kubernetes \
        $DOCKER_RUNNER \$@
EOF
    chmod 755 $KUBEADM_RUNNER
}

# install a runner for kubeadm
install_kubeadm_runner() {
    script=$(cat <<-EOF
#!/bin/sh

[ -d /etc/kubernetes/manifests ] && rm -rf /etc/kubernetes/manifests/*
mkdir -p /etc/kubernetes/manifests

echo "Killing any kubeadm running..."
killall kubeadm &>/dev/null || /bin/true

echo "Resetting any previous setup..."
sudo $1 reset

echo "Running $1 in the background..."
nohup $1 \$@ &
sleep 2

EOF
)

	if [ -x $1 ] ; then
	    log "creating a runner for $1"
	    echo "$script" >$KUBEADM_RUNNER
        chmod 755 $KUBEADM_RUNNER
	else
	    log "WARNING: did not find kubeadm at $KUBEADM_RUNNER"
	fi
}

##########################################################################################

log "removing previous link at $KUBEADM_RUNNER"
rm -f $KUBEADM_RUNNER

# there are two ways we can identify the distro: with the help of lsb-release, or
# with some key files in /etc (like /etc/debian_version)
if [ -x $LSB_RELEASE ] ; then
	ID=$($LSB_RELEASE --short --id)
	case $ID in
	RedHatEnterpriseServer|CentOS|Fedora)
		RELEASE=$($LSB_RELEASE --short --release | cut -d. -f1)
		install_yum
		;;

	Ubuntu|Debian)
		install_apt
		;;

	*SUSE*)
		desc=$($LSB_RELEASE --short --description)
		RELEASE=$($LSB_RELEASE --short --release)
		case $desc in
		*openSUSE*)
		    DIST=opensuse$RELEASE
		    ;;
		*Enterprise*)
		    DIST=sles$RELEASE
		    ;;
		esac
		install_zypper
		;;

	*)
		log "could not get the release/distribution from $LSB_RELEASE"
		DIST=
		;;
	esac
else
	if [ -f /etc/debian_version ] ; then
		install_apt
	elif [ -f /etc/fedora-release ] ; then
		install_yum
	elif [ -f /etc/redhat-release ] ; then
		install_yum
	elif [ -f /etc/SuSE-release ] ; then
		install_zypper
	elif [ -f /etc/centos-release ] ; then
		install_yum
	fi
fi

[ -x $KUBEADM_RUNNER ] || install_docker_runner
