#!/bin/sh

##########################################################################################
# kubeadm setup script
##########################################################################################

LSB_RELEASE=/usr/bin/lsb_release

# packages version to install
K8S_VERSION="1.5.5"

# the executable that packages will install, and the packages per distro
KUBEADM_EXE=/usr/bin/kubeadm
KUBEADM_PKG_SUSE=kubernetes-kubeadm
KUBEADM_PKG_APT=kubeadm
KUBEADM_PKG_YUM=kubeadm

# we will try to discover the DIST and RELEASE
ID=
DIST=
RELEASE=

# a container that can be used for running kubeadm
DOCKER_RUNNER_IMAGE="inercia/kubeadm"

##########################################################################################

log()    { echo "[kubeadm setup] $@" ;     }
abort()  { log "$@" ; exit 1 ; }

kill_kubeadm() {
    local this=$(basename $0)
    local pid=$(ps ax | grep -v $this | grep "kubeadm" | grep -v grep | cut -f2 -d' ')
    if [ -n "$PID" ] ; then
        log "killing a previous kubeadm running with PID=$pid"
        kill "$pid" &>/dev/null || /bin/true
    fi
}

add_zypper_repo() {
    local name="$1"
    local url="$2"

	if [ ! -f "/etc/zypp/repos.d/$name.repo" ] ; then
		log "adding repository $name"
        zypper $zypper_args ar -Gf "$url" $name || abort "could not enable $name repo"
		zypper $zypper_args ref "$name"         || abort "could not refresh the $name repo"
	else
		log "repository $name already found: skipping installation of the repo"
	fi
}

##########################################################################################

# installation for SUSE variants: OpenSUSE/SLE/CaaSP...
install_zypper() {
    local zypper_args="-n --no-gpg-checks --quiet --no-color"

    local extra_repo_name="extra"
    local extra_repo_url="http://download.opensuse.org/repositories/home:/asaurin:/branches:/Virtualization:/containers/openSUSE_Leap_42.2/"

    local packages="$KUBEADM_PKG_SUSE==$K8S_VERSION kubernetes-kubelet==$K8S_VERSION kubernetes-client==$K8S_VERSION"

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
    add_zypper_repo "containers"       "$containers_repo_url"
    add_zypper_repo "$extra_repo_name" "$extra_repo_url"

    log "checking we have everything we need..."
	zypper $zypper_args in -y $packages || abort "could not finish the installation of packages"
    log "... everything installed"

    log "allowing privileged conatiners in kubernetes config"
    cat <<EOF > /etc/kubernetes/config
KUBE_LOGTOSTDERR="--logtostderr=true"
KUBE_LOG_LEVEL="--v=2"
KUBE_ALLOW_PRIV="--allow-privileged=true"
KUBE_MASTER=""
EOF

    log "starting services"
    systemctl enable docker   && systemctl start docker    || abort "could not start docker"
    systemctl enable kubelet  && systemctl start kubelet   || abort "could not start kubelet"
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
    [ -x $KUBEADM_EXE     ] || yum install -y $KUBEADM_PKG_YUM       || abort "could not finish the installation of kubeadm"
    [ -x /usr/bin/kubelet ] || yum install -y kubelet kubernetes-cni || abort "could not finish the installation of kubelet"
    [ -x /usr/bin/docker  ] || yum install -y docker                 || abort "could not finish the installation of docker"
    [ -x /usr/bin/kubectl ] || yum install -y kubectl                || abort "could not finish the installation of kubectl"
    log "... everything installed"

    log "starting services"
    systemctl enable docker  && systemctl start docker || abort "could not start docker"
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
    [ -x $KUBEADM_EXE     ] || apt-get install -y $KUBEADM_PKG_APT       || abort "could not finish the installation of kubeadm"
    [ -x /usr/bin/kubelet ] || apt-get install -y kubelet kubernetes-cni || abort "could not finish the installation of kubelet"
    [ -x /usr/bin/docker  ] || apt-get install -y docker.io              || abort "could not finish the installation of docker"
    [ -x /usr/bin/kubectl ] || apt-get install -y kubectl                || abort "could not finish the installation of kubectl"
    log "... everything installed"
}

# install a script for running kubeadm inside a container
# TODO: this is not functional yet!!!
install_docker_runner() {
    log "WARNING: falling back to the docker runnner for kubeadm !!!"
    local runner="/tmp/kubeadm-docker-runner.sh"
	cat <<EOF >"$runner"
#!/bin/sh
docker run --rm -ti --privileged --net=host \
        -v /etc/kubernetes:/etc/kubernetes \
        $DOCKER_RUNNER_IMAGE \$@
EOF
    chmod 755 "$runner"
    KUBEADM_EXE="$runner"
}

##########################################################################################

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

# TODO: fallback to a docker runner if installation fails...
# [ -x $KUBEADM_EXE ] || install_docker_runner

[ -x $KUBEADM_EXE ] || abort "no kubeadm executable available at $KUBEADM_EXE"
