# kubeadm provider and provisioner

The kubeadm provider is used for interacting with kubeadm for creating Kubernetes clusters.

## Example Usage

```hcl
resource "kubeadm" "main" {
  api {
    external = "loadbalancer.external.com"
  }
  
  network {
    dns_domain = "my_cluster.local"  
    services = "10.25.0.0/16"
  }
}

# from the libvirt provider
resource "libvirt_domain" "master" {
  name = "master"
  memory = 1024
  ...
  provisioner "kubeadm" {
    config = "${kubeadm.main.config}"
  }
}

# from the libvirt provider
resource "libvirt_domain" "minion" {
  count      = 3
  name       = "minion${count.index}"
  ...
  provisioner "kubeadm" {
    config = "${kubeadm.main.config}"
    join = "${libvirt_domain.master.network_interface.0.addresses.0}"
  }
}
```

## Contents

* [Installation](Installation) instructions.
* Using `kubeadm` in your Terraform scripts:
  * The [`resource "kubeadm"`](Resource_kubeadm) configuration block.
  * The [`provisioner "kubeadm"`](Provisioner_kubeadm) block.
  * [Additional tasks](Additional_tasks) necessary for having a
  fully functional Kubernetes cluster, like installing some Pods
  Security Policy...
* [Roadmap, TODO and vision](Roadmap).
* [Examples](examples/README.md) for several providers like
_libvirt_, _LXD_, _AWS_, etc.
