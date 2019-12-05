#!/bin/sh

# Upgrading Helm also upgrades the kubernetes dependency, run this script in order to fix all the
# unknown revision 0.0v.0 errors

# Adapted from here https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-521493597
# didn't work for me out of the box
# TODO fix the original script substitutions
VERSION=${1}
if [ -z "$VERSION" ]; then
    echo "Must specify version! For example 1.16.3"
    exit 1
fi
MODS=$(
    curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v${VERSION}/go.mod |
    sed -n 's|.*k8s.io/\(.*\) => ./staging/src/k8s.io/.*|k8s.io/\1|p'
)
echo $MODS
for MOD in $MODS; do
    V=$(
        go mod download -json "${MOD}@kubernetes-${VERSION}" |
        sed -n 's|.*"Version": "\(.*\)".*|\1|p'
    )
    go mod edit "-replace=${MOD}=${MOD}@${V}"
done
go get "k8s.io/kubernetes@v${VERSION}"
