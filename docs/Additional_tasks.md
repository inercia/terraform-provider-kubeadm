# Additional tasks

## CNI

* `kubeadm` currently does not install any CNI driver
(ie, `flannel`,`calico`, etc). However, you can use the `manifests`
in the `provisioner` for loading your preferred CNI manifest once the first master is ready.

```hcl
variable "manifests" {
  type = "list"
  default = [
    "https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml",
  ]
  description = "List of manifests to load after setting up the first master"
}

resource "..." "master" {
  provisioner "kubeadm" {
    config     = "${data.kubeadm.main.config.init}"
    kubeconfig = "${var.kubeconfig}"
    manifests  = "${var.manifests}"
  }
}
```
