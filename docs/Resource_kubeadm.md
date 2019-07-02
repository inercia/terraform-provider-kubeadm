# kubeadm resource

The resource provides the global configuration for the cluster.

## Example Usage

```hcl
resource "kubeadm" "main" {
  # the kubeconfig file created
  config_path = "/home/myself/.kube/config"
  
  api {
    # the address used byy our external load balancer
    external = "loadbalancer.external.com"
  }
  
  network {
    dns_domain = "my_cluster.local"  
    services = "10.25.0.0/16"
  }
}
```

## Argument Reference

The following arguments are supported:

* `config_path` - The local copy of the `kubeconfig` that will
be created after bootstrapping the cluster. This file can be used in
the `--config` argument of `kubectl` for managing the cluster with
administrative privileges. 
  * NOTE: any previous `config_path` file will be moved to a `.bak` file
  at the beginning of the cluster bootstrap, regardless of the success/failure
  of the operation.
* `addons` - (Optional) Addons to deploy (see section below).
* `api` - (Optional) API server configuration (see section below).
* `certs` - (Optional) user-provided certificates (see section below).
* `cni` - (Optional) CNI configuration (see section below).
* `network` - (Optional) network configuration (see section below).
* `images`  - (Optional) images used for running the different services (see section below).
* `etcd`  - (Optional) `etcd` configuration (see section below).
* `version`  - (Optional) kubernetes version.

## Nested Blocks

### `addons`

#### Arguments

* `dashboard` - (Optional) when `true`, deploy the Kubernetes Dashboard.
* `helm` - (Optional) when `true`, deploy Helm.

### `api`

#### Arguments

* `external` - (Optional) stable IP/DNS (and port) for the control plane
(for example, the load balancer, or some DNS name). This name or address
will be included in the certificates gegnerated for the API server, so
HTTPS connections will not fail.
  * NOTE: **IMPORTANT**: an external, stable IP/DNS  is required in order
  to support multiple masters. And once the cluster is created, this
  parameter cannot be changed (that would trigger a cluster recreation). So
  you must realize that, if you leave this argument empty, your cluster
  will never grow the number of masters. 
* `internal` - (Optional) IP/DNS and port the local API server advertises
it's accessible.
* `alt_names` - (Optional) list of SANs to use in api-server certificate.
Example: `IP=127.0.0.1,IP=127.0.0.2,DNS=localhost`, If empty, SANs will
be obtained from the _external_ and _internal_ names/IPs.

### `cni`

#### Arguments

* `plugin` - (Optional) when not empty, name of the CNI plugin to load in the
cluster after the initial bootstrap. There is a list of pre-defined manifests
to load for some well-known plugins, being the list of recognized names:
  * `flannel`
* `plugin_manifest`  - (Optional) when not empty, load the CNI driver by using
the provided manifest. When both `plugin` and `plugin_manifest` are provided,
the former one is ignored.
* `bin_dir` - (Optional) binaries directory for CNI.
* `conf_dir` - (Optional) configuration directory for CNI.

### `certs`

#### Arguments

Example:

```hcl
        
resource "kubeadm" "k8s" {
  config_path = "/tmp/kubeconfig"

  # provided a specific CA certificate/key inlined in the "certs" block
  certs {
	ca_crt =<<EOF
-----BEGIN CERTIFICATE-----
MIICwjCCAaqgAwIBAgIBADANBgkqhkiG9w0BAQsFADASMRAwDgYDVQQDEwdldGNk
LWNhMB4XDTE5MDYyODE1NTM1M1oXDTI5MDYyNTE1NTM1M1owEjEQMA4GA1UEAxMH
...
QasRiemsP8IWvOwcKGViNUC2Ag5EEh8S8PMlLP+3/pzkEPfIHgk=
-----END CERTIFICATE-----
EOF

	ca_key =<<EOF
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAzU/cJiB3/Fr85gW6dpCbDQTyZpVuB8LtyfwFHhsUrCpVJ/U0
B4lfH2n8E8VB62SeGtaXcbYnScNgkaDgQ+SvkHlIDjp16Z3cFjHLOUyF3cVOBbDs
...
OflxkYMD4H/BuT3uuX4BR0Ko32wAyNn/AJmgekiPjQ/NGfwG0CS2fGY= 
-----END RSA PRIVATE KEY-----
EOF
  }
}
```

* `ca_crt` - (Optional) user-provided CA certificate.
* `ca_key` - (Optional) user-provided CA key.
* `sa_crt` - (Optional) user-provided Service Account certificate.
* `sa_key` - (Optional) user-provided Service Account key.
* `etcd_crt` - (Optional) user-provided `etcd` certificate.
* `etcd_key` - (Optional) user-provided `etcd` key.
* `proxy_crt` - (Optional) user-provided front-proxy certificate.
* `proxy_key`- (Optional) user-provided front-proxy key.

Note well: all these certificates are optional: they will be generated  automatically
if not provided.

### `network`

#### Arguments

Example:
```hcl
resource "kubeadm" "main" {
  network {
    dns_domain = "mycluster.com"
    services   = "10.25.0.0/16"
  }
}
```

* `services` - (Optional) subnet used by k8s services. Defaults to `10.96.0.0/12`.
* `pods` - (Optional) subnet used by pods.
* `dns_domain` - (Optional) DNS domain used by k8s services. Defaults to `cluster.local`.

### `images`

#### Arguments

* `kube_repo` - (Optional) the kubernetes images repository.
* `etcd_repo` - (Optional) the etcd image repository.
* `etcd_version` - (Optional) the etcd version.

### `etcd`

#### Arguments

Example:
```hcl
resource "kubeadm" "main" {
  etcd {
    endpoints = ["server1.com:2379", "server2.com:2379"]
  }
}
```

* `endpoints` - (Optional) list of etcd servers URLs, as `host:port`.

### `runtime`

#### Arguments

* `engine` - (Optional) containers runtime to use: `docker`/`crio`.
* `extra_args` - (Optional) extra arguments for the components:
  * `api_server` - (Optional) extra arguments for the API server.
  * `controller_manager` - (Optional) extra arguments for the controller manager.
  * `scheduler` - (Optional) extra arguments for the scheduler.
  * `kubelet` - (Optional) extra arguments for the kubelet.

Example:
```hcl
resource "kubeadm" "main" {
  runtime {
    engine = "crio"
    extra_args {
      api_server = {
        "feature-gates" = "DynamicKubeletConfig=true"
      }
    }
  }
```


## Attributes Reference

The following attributes are exported:


* `config` - a dictionary with some config exported to the provisioners,
but can also be directly accessible in case you need it.
  * `init` - a valid `kubeadm` init configuration file (encoded with `base64`)
  ready for doing a `kubeadm init`.
  * `join` - a valid `kubeadm` join configuration file (encoded with `base64`)
  ready for doing a `kubeadm join` and joining the cluster. This can be useful
  for joining the cluster _a posteriori_ without the intervention of Terraform. 
  For example, you can prepare some `cloud-init` configuration file for
  launching automatically new machines in some autoscaling group, with 
  something like:
    ```hcl
    data "template_file" "script" {
      template = <<-EOT
      # write a config file ready for doing a `kubeadm join` 
      write_files:
        -   encoding:    b64
            content:     ${kubeadm_config}
            owner:       root:root
            path:        /etc/kubernetes/kubeadm.conf
            permissions: '0644'
      # join the cluster on the first boot
      # (we assume kubeadm is already available in the VM image)
      bootcmd:
        - kubeadm join --config=/etc/kubernetes/kubeadm.conf
      EOT
    
      vars {
        # NOTE: we don't need to do a "${base64decode(kubeadm.main.config.init)}"
        # beacuse cloud-init can decode base64 for us.
        kubeadm_config = "${kubeadm.main.config.init}"
      }
    }
    ```
  * `ca_crt`
  * `ca_key`
  * `sa_crt`
  * `sa_key`
  * `etcd_crt`
  * `etcd_key`
  * `proxy_crt`
  * `proxy_key` - certificates generated for the kubernetes cluster. They can be
  used in some other Terraform resources, for example you could use the certificate
  generated for the front proxy and assign it to the AWS load balancer:
      ```hcl
      resource "aws_iam_server_certificate" "front-proxy" {
        name             = "front-proxy"
        certificate_body = "${kubeadm.main.config.proxy_crt}"
        private_key      = "${kubeadm.main.config.proxy_key}"
      }
    
      resource "aws_elb" "my-application" {
        name                      = "terraform-asg-deployment-example"
        availability_zones        = ["us-west-2a"]
        cross_zone_load_balancing = true
      
        listener {
          instance_port      = 80
          instance_protocol  = "http"
          lb_port            = 443
          lb_protocol        = "https"
          ssl_certificate_id = "${aws_iam_server_certificate.front-proxy.arn}"
        }
      }
      ```

