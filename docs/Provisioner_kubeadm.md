# kubeadm provisioner

The resource provides the global configuration for the cluster.

## Example Usage

```hcl
# from the libvirt provider
resource "libvirt_domain" "minion" {
  count      = 3
  name       = "minion${count.index}"
  ...
  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config.join}"
    join = "${libvirt_domain.master.network_interface.0.addresses.0}"
  }
}
```

## Argument Reference

  * `config` - a reference to the `config` configuration of the _provider_.
  * `join` - (Optional) the address of the node to join in the cluster. 
  The absence of a `join` indicates that this node will be used as a kubernetes
  master and seeder for the cluster.
  * `install` - (Optional) options for the autoinstaller for dependencies.
  * `prevent_sudo` - (Optional) prevent the usage of `sudo` for running commands.
  * `manifests` - (Optional) list of extra manifests to `kubectl apply -f`
  in the first master after the API server is upp and running.

## Nested Blocks

### `install`

#### Arguments

* `auto` - (Optional) try to automatically install kubeadm with the built-in helper script.
* `script` - (Optional) user-provided installation script.
* `version` - (Optional) kubeadm version to install.

## Known limitations

* The `kubeadm-setup.sh` tries to does its best in order to install
`kubeadm`, but some distros have not been tested too much. I'use
used `libvirt` with _OpenSUSE Leap 42.2_ images for running my
tests, so that could be considered the perfect combination for
trying this...
