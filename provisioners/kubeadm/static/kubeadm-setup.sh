#!/bin/sh

##########################################################################################
# kubeadm setup script
##########################################################################################

LSB_RELEASE=/usr/bin/lsb_release

# the executable that packages will install, and the packages per distro
KUBEADM_EXE=/usr/bin/kubeadm

KUBEADM_PKG_SUSE="kubernetes-kubeadm"
KUBEADM_PKG_SUSE_VERS="1.5.5"
KUBEADM_PKG_SUSE_REPO="http://download.opensuse.org/repositories/home:/asaurin:/branches:/Virtualization:/containers/openSUSE_Leap_42.2/"

KUBEADM_PKG_APT="kubeadm"
KUBEADM_PKG_APT_VERS="1.5.4"
KUBEADM_PKG_APT_REPO="http://apt.kubernetes.io/"

KUBEADM_PKG_YUM="kubeadm"
KUBEADM_PKG_YUM_VERS="1.5.4"

# we will try to discover the DIST and RELEASE
ID=
DIST=
RELEASE=

# a container that can be used for running kubeadm
DOCKER_RUNNER_IMAGE="inercia/kubeadm"

##########################################################################################

log()    { echo "[kubeadm setup] $@" ;     }
abort()  { log "$@" ; exit 1 ; }

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

restart_services() {
    log "starting services"
    systemctl enable docker   && systemctl start docker  || abort "could not start docker"
    systemctl enable kubelet  && systemctl start kubelet || abort "could not start kubelet"
}

##########################################################################################

# installation for SUSE variants: OpenSUSE/SLE/CaaSP...
install_zypper() {
    local zypper_args="-n --no-gpg-checks --quiet --no-color"

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
    add_zypper_repo "containers"  "$containers_repo_url"
    add_zypper_repo "extra"       "$KUBEADM_PKG_SUSE_REPO"

    log "checking we have everything we need..."
    local packages="$KUBEADM_PKG_SUSE==$KUBEADM_PKG_SUSE_VERS kubernetes-kubelet==$KUBEADM_PKG_SUSE_VERS kubernetes-client==$KUBEADM_PKG_SUSE_VERS"
	zypper $zypper_args in -y $packages || abort "could not finish the installation of packages"
    log "... everything installed"

    log "allowing privileged conatiners in kubernetes config"
    cat <<EOF > /etc/kubernetes/config
KUBE_LOGTOSTDERR="--logtostderr=true"
KUBE_LOG_LEVEL="--v=2"
KUBE_ALLOW_PRIV="--allow-privileged=true"
KUBE_MASTER=""
EOF

    restart_services
}

# installation for RedHat variants: RedHat/CentOS...
install_yum() {
	log "Installing for RedHat..."
	if [ ! -f /etc/yum.repos.d/kubernetes.repo ] ; then
		[ -n "$RELEASE" ] || abort "can't continue without knowing the release"
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
	local packages="$KUBEADM_PKG_YUM-$KUBEADM_PKG_YUM_VERS kubelet kubernetes-cni docker kubectl"
    [ -x $KUBEADM_EXE ] || yum install -y $packages || abort "could not finish the installation of kubeadm"
    log "... everything installed"

	restart_services
}

# installation for Debian variants: debian/Ubuntu...
install_apt() {
	log "installing for Ubuntu|Debian..."
	apt-get update && apt-get install -y apt-transport-https
	if [ ! -f /etc/apt/sources.list.d/kubernetes.list ] ; then
		curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
		echo "deb $KUBEADM_PKG_APT_REPO kubernetes-xenial main" >> /etc/apt/sources.list.d/kubernetes.list
		apt-get update
	else
		log "repository already found: skipping installation of the repo"
	fi

    log "checking we have everything we need..."
    local packages="$KUBEADM_PKG_APT=$KUBEADM_PKG_APT_VERS kubelet kubernetes-cni docker.io kubectl"
    [ -x $KUBEADM_EXE ] || apt-get install -y $packages || abort "could not finish the installation of kubectl"
    log "... everything installed"

	restart_services
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
