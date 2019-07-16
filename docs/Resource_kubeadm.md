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
* `cloud` - (Optional) cloud provider configuration (see section below).
* `cni` - (Optional) CNI configuration (see section below).
* `etcd`  - (Optional) `etcd` configuration (see section below).
* `images`  - (Optional) images used for running the different services (see section below).
* `network` - (Optional) network configuration (see section below).
* `runtime` - (Optional) runtime and operational configuration (see section below).
* `version`  - (Optional) kubernetes version.

## Nested Blocks

### `addons`

The `addons` block provides flags for enabling/disabling and configuring
different addons that can be deployed to the cluster.

#### Arguments

* `dashboard` - (Optional) when `true`, deploy the Kubernetes Dashboard.
* `helm` - (Optional) when `true`, deploy Helm.

### `api`

The `api` block provided different configuration options for the API server.

Example:

```hcl
resource "kubeadm" "main" {
  # ...
  api {
    # use the load balancer's address as an external addrerss for the API server
    external = "my-lb.my-company.com"

    # some other names to include in the cerificate that will be generated
    alt_names = "IP=193.144.60.101,DNS=server.my-company.com"
  }
}
```

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

The `cni` block is used for configuring the CNI plugin.

Example:

```hcl
resource "kubeadm" "k8s" {
  # ...
  cni {
    plugin = "flannel"

    # use a non-standard directory
    bin_dir = "/usr/lib/cni"

    flannel {
      # use some specific backend 
      backend = "host-gw"
    }
  }
}


```
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
* `flannel`  - (Optional) Flannel configuration options:
  * `version` - (Optional) the flannel image version.
  * `backend` - (Optional) Flannel backend: `vxlan`, `host-gw`, 
  `udp`, `ali-vpc`, `aws-vpc`, `gce`, `ipip`, `ipsec`.

### `certs`

The `certs` block can be used for providing specific certificates instead of
relaying in the automatically generated ones. 

Example, using some inlined certificates:

```hcl
        
resource "kubeadm" "k8s" {
  config_path = "/tmp/kubeconfig"

  # provided a specific CA certificate/key inlined in the "certs" block
  # could also be read from some file with something like "${file("somepath/ca.crt")}"
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

#### Arguments

* `ca_crt` - (Optional) user-provided CA certificate.
* `ca_key` - (Optional) user-provided CA key.
* `sa_crt` - (Optional) user-provided Service Account certificate.
* `sa_key` - (Optional) user-provided Service Account key.
* `etcd_crt` - (Optional) user-provided `etcd` certificate.
* `etcd_key` - (Optional) user-provided `etcd` key.
* `proxy_crt` - (Optional) user-provided front-proxy certificate.
* `proxy_key`- (Optional) user-provided front-proxy key.

All these certificates are completely optional: they will be generated
automatically by the `kubeadm` resource if not provided. However, in some cases
it is useful to provide certificates from other resources in your Terraform script.

For example, you could also generate a certifciate with Terraform and share it in
different parts of your code:

```hcl
resource "tls_private_key" "ca" {
  algorithm = "ECDSA"
}

resource "tls_self_signed_cert" "ca" {
  key_algorithm   = "${tls_private_key.example.algorithm}"
  private_key_pem = "${tls_private_key.example.private_key_pem}"

  # Reasonable set of uses for a server SSL certificate.
  allowed_uses = [
      "key_encipherment",
      "digital_signature",
      "server_auth",
  ]

  dns_names = ["my-company.com", "my-company.net"]

  subject {
      common_name  = "my-company.com"
      organization = "My Company, Inc"
  }
}

resource "kubeadm" "k8s" {
  config_path = "/tmp/kubeconfig"

  certs {
	ca_crt = "${tls_self_signed_cert.ca.cert_pem}"

	ca_key = "${tls_private_key.ca.private_key_pem}"
  }
}
```

Notes:
  * Changes in the certificates, for example after a certificate rotation, will
  currently invalidate the kubeadm resources and, as a consequence, recreate
  the cluster. It is not recommended to rely on external resources for rotating
  certifciates and to [use kubeadm for rotating certificates](https://kubernetes.io/docs/tasks/administer-cluster/kubeadm/kubeadm-certs/). 
  

### `network`

The `network` block is used for configuring the network.

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

The `images` block provides a way for changing the images used for running
the control plane in the cluster.

#### Arguments

* `kube_repo` - (Optional) the kubernetes images repository.
* `etcd_repo` - (Optional) the etcd image repository.
* `etcd_version` - (Optional) the etcd version.

### `etcd`

The `etcd` block can be used for using an external etcd cluster, providing
the endpoints that will be used.

Example:

```hcl
resource "kubeadm" "main" {
  etcd {
    endpoints = ["server1.com:2379", "server2.com:2379"]
  }
}
```

#### Arguments

* `endpoints` - (Optional) list of etcd servers URLs, as `host:port`.


### `cloud`

The `cloud` block provides some configuration for  the cloud provider.

Example:

```hcl
resource "kubeadm" "main" {
 cloud {
   provider = "openstack"
   manager_flags = "--allocate-node-cidrs=true --configure-cloud-routes=true --cluster-cidr=172.17.0.0/16"
   config =<<EOF
[Global]
username=user
password=pass
auth-url=https://<keystone_ip>/identity/v3
tenant-id=c869168a828847f39f7f06edd7305637
domain-id=2a73b8f597c04551a0fdc8e95544be8a

[LoadBalancer]
subnet-id=6937f8fa-858d-4bc9-a3a5-18d2c957166a
EOF       
 }
}
```

#### Arguments

* `provider` - (Optional) the [cloud provider](https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/)
to use. Can be `aws`, `azure`, `cloudstack`, `gce`, `openstack`, etc.
* `manager_flags` - (Optional) some additional flags for the cloud provider manager.
* `config` - (Optional) the Cloud Provider configuration. This can be read from a file
(with something like `file("${path.module}/cloud.conf")`), from a `template` or provided 
inline with a _heredoc_ block.

### `runtime`

The `runtime` block provides some operational configuration for different components
of the Kubernetes cluster.

Example:

```hcl
resource "kubeadm" "main" {
  runtime {
    engine = "crio"
    extra_args {
      api_server = {
        # this will be transaleted to a "--feature-gates=DynamicKubeletConfig=true" argument
        "feature-gates" = "DynamicKubeletConfig=true"
      }
    }
  }
```

#### Arguments

* `engine` - (Optional) containers runtime to use: `docker`/`crio`.
* `extra_args` - (Optional) maps with extra arguments for the components:
  * `api_server` - (Optional) map with extra arguments for the API server.
  * `controller_manager` - (Optional) map with extra arguments for the controller manager.
  * `scheduler` - (Optional) map with extra arguments for the scheduler.
  * `kubelet` - (Optional) map with extra arguments for the kubelet.

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
      template = <<EOT
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
  * `cloud_provider`, `cloud_provider_flags`, `cloud_config` - the cloud
  provider configuration. 
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

