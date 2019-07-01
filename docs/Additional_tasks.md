# Additional tasks

Once you have created the cluster with Terraform you should do some other tasks in order
to have a production-quality cluster:

* create some [Pod Security Policy](https://kubernetes.io/docs/concepts/policy/pod-security-policy/)
and apply it before running any workload in the cluster. 
* if the CNI plugin supports it, apply some
[Network Security Policy](https://kubernetes.io/docs/concepts/services-networking/network-policies/).
* install [Dex](https://github.com/dexidp/dex/blob/master/Documentation/kubernetes.md)
for authentication, and connect it to your LDAP servers, SAML providers, or some
identity provider like GitHub, Google, and Active Directory. Do not distribute the `kubeconfig`
file created for managing this cluster.   
