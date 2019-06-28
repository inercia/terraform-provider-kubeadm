# kubeadm provisioner

The kubeadm provisioner is responsible for starting `kubeadm` with the right
parameters for configuring the machine as part of the kubernetes cluster.

## Example Usage

For provisioning a machine as a _master_ in the cluster:

```hcl
# from the libvirt provider
resource "libvirt_domain" "minion" {
  name       = "master"
  ...
  provisioner "kubeadm" {
    config = "${kubeadm.main.config}"
    install {
      auto = true
    }
  }
}
```

and for provisioning some _worker_ nodes:

```hcl
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

## Argument Reference

  * `role` - (Optional) defines the role of the machine: `master` or `worker`.
  If `join` is empty, it defaults to the `master` role, otherwise it defaults
  to the `worker` role. 
  * `config` - a reference to the `kubeadm.<resource-name>.config` attribute of the _provider_.
  * `join` - (Optional) the address (either a resolvable DNS name or an IP) of the
  node in the cluster to join. The absence of a `join` indicates that this node 
  will be used for bootstrapping the cluster and will be the seeder for the other
  nodes of the cluster. When `join` is not empty and `role` is `master`, the node
  will join the cluster's Control Plane.
  * `install` - (Optional) options for the autoinstaller script (see section below).
  * `prevent_sudo` - (Optional) prevent the usage of `sudo` for running commands.
  * `manifests` - (Optional) list of extra manifests to `kubectl apply -f`
  in the booststrap master after the API server is up and running. These manifests
  can be either local files or URLs.
  * `nodename` - (Optional) name for the `.Metadata.Name` field of the Node API
  object that will be created in this `kubeadm init` or `kubeadm join` operation.
  This is also used in the CommonName field of the kubelet's client certificate
  to the API server. Defaults to the hostname of the node if not provided.
  * `ignore_checks` - (Optional) list of kubeadm preflight checks to ignore
  when provisioning. Example:
    ```hcl
    ignore_checks = [
      "NumCPU",
      "FileContent--proc-sys-net-bridge-bridge-nf-call-iptables",
      "Swap",
    ]
    ```

## Nested Blocks

### `install`

Example:

```hcl
resource "libvirt_domain" "master" {
  name       = "master${count.index}"
  ...
  provisioner "kubeadm" {
    config = "${kubeadm.main.config}"
    install {
      # try to install `kubeadm` automatically with the builin script
      auto = true
    }
  }
}
```

#### Arguments

* `auto` - (Optional) try to automatically install kubeadm with
[the built-in helper script](https://github.com/inercia/terraform-provider-kubeadm/blob/master/internal/assets/static/kubeadm-setup.sh).
* `script` - (Optional) a user-provided installation script. It should install `kubeadm`
in some directory available in the default `$PATH`.
* `inline` - (Optional) some inline code for installing kubeadm in the remote machine. Example:
    ```hcl
    resource "libvirt_domain" "master" {
      name       = "master${count.index}"
      ...
      provisioner "kubeadm" {
        config = "${kubeadm.main.config}"
        install {
          auto = true
          inline = <<-EOT
          #!/bin/sh
          RELEASE="$(curl -sSL https://dl.k8s.io/release/stable.txt)"
          mkdir -p /opt/bin
          cd /opt/bin
          curl -L --remote-name-all https://storage.googleapis.com/kubernetes-release/release/${RELEASE}/bin/linux/amd64/{kubeadm,kubelet,kubectl}
          chmod +x {kubeadm,kubelet,kubectl}
          EOT
        }
      }
    }
    ```
* `version` - (Optional) kubeadm version to install by the auto-installation script.
    * NOTE: this can be ignored by the auto-install script in some OSes
    where there are not so many installation alternatives.

### Known limitations

* The `kubeadm-setup.sh` tries to does its best in order to install
`kubeadm`, but some distros have not been tested too much. I've
used `libvirt` with _OpenSUSE Leap_ images for running my
tests, so that could be considered the perfect combination for
trying this...

