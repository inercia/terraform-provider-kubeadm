# Roadmap and TODO

* [ ] Multi-master, as described [here](https://kubernetes.io/docs/setup/independent/high-availability/#stacked-control-plane-and-etcd-nodes).
* [ ] Node removal on destruction, cordoning the node when the
`provisioner` detects that the underlying resource is being destroyed.
* [ ] Support adding new nodes well after the seeder was created
by
  1) loading an existing `kubeconfig`
  2) using the current token if still valid, or creating a new token otherwise
* [ ] The ability to specify the Cloud Provider.
