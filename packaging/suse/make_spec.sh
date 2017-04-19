#!/bin/bash

if [ -z "$1" ]; then
  cat <<EOF
usage:
  ./make_spec.sh PACKAGE
EOF
  exit 1
fi

cd $(dirname $0)

YEAR=$(date +%Y)
VERSION=$(cat ../../VERSION)
COMMIT_UNIX_TIME=$(git show -s --format=%ct)
VERSION="${VERSION%+*}+$(date -d @$COMMIT_UNIX_TIME +%Y%m%d).$(git rev-parse --short HEAD)"
NAME=$1
GITREPONAME=$(basename `git rev-parse --show-toplevel`)

cat <<EOF > ${NAME}.spec
#
# spec file for package $NAME
#
# Copyright (c) $YEAR SUSE LINUX GmbH, Nuernberg, Germany.
#
# All modifications and additions to the file contributed by third parties
# remain the property of their copyright owners, unless otherwise agreed
# upon. The license for this file, and modifications and additions to the
# file, is the same license as for the pristine package itself (unless the
# license for the pristine package is not an Open Source License, in which
# case the license is the MIT License). An "Open Source License" is a
# license that conforms to the Open Source Definition (Version 1.9)
# published by the Open Source Initiative.

# Please submit bugfixes or comments via http://bugs.opensuse.org/
#

# Make sure that the binary is not getting stripped.
%{go_nostrip}

Name:           $NAME
Version:        $VERSION
Release:        0
License:        MPL-2.0
Summary:        Experimental Terraform plugin for kubeadm
Url:            https://github.com/inercia/terraform-kubeadm/
Group:          System/Management
Source:         %{name}-%{version}.tar.xz
BuildRoot:      %{_tmppath}/%{name}-%{version}-build

BuildRequires:  golang-packaging
BuildRequires:  libvirt-devel
BuildRequires:  xz
BuildRequires:  go >= 1.6

Requires:       terraform >= 0.8.5
Requires:       genisoimage
%{go_provides}

%description
Terraform plugin for using kubeadm for creating kubernetes clusters.

%prep
%setup -q -n %{name}-%{version}

%build
%goprep github.com/inercia/terraform-kubeadm
echo ">>>> making"
export GOPATH=%{_builddir}/go
export GOBIN=$GOPATH/bin
make -C $GOPATH/src/github.com/inercia/terraform-kubeadm

%install

echo ">>>> installing"
install -m 755 -d %{buildroot}%{_bindir}
install -p -m 755 -t %{buildroot}%{_bindir} %{_builddir}/go/bin/terraform-{provider,provisioner}-kubeadm

rm -rf %{buildroot}/%{_libdir}/go/contrib

%files
%defattr(-,root,root,-)
%doc README.md
%{_bindir}/terraform-{provider,provisioner}-kubeadm

%changelog
EOF
