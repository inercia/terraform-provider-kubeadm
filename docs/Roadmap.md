# Roadmap and TODO

* [ ] Node removal on destruction, cordoning the node when the
`provisioner` detects that the underlying resource is being destroyed.
* [ ] Support adding new nodes well after the seeder was created
by
  1) loading an existing `kubeconfig`
  2) using the current token if still valid, or creating a new token otherwise
* [ ] The ability to specify the Cloud Provider.
* [ ] The ability to customize the CNI driver (ie, change the Flannel backend).
* [ ] The ability to load some PSP.
* [ ] Publish the provider in the [community page](https://www.terraform.io/docs/providers/type/community-index.html).
