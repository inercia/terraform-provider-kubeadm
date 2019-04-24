# Installation

## From source

1.  `go get -d github.com/inercia/terraform-provider-kubeadm`

2.  Make sure your Terraform binary has been built with some stable version,
    otherwise you will get a
    `Incompatible API version with plugin. Plugin version: 1, Ours: 2`
    error at runtime. If you built it from sources:
    ```
    cd $GOPATH/src/github.com/hashicorp/terraform
    git checkout v0.8.0
    cd $GOPATH/src/github.com/inercia/terraform-provider-kubeadm
    ```
3.  Run `make` to build the binaries. You will now find two binaries
at your `$HOME/.terraform.d/plugins` directory:
    ```
    ls $HOME/.terraform.d/plugins
    terraform-provider-kubeadm  terraform-provisioner-kubeadm
    ```
