# Roadmap and TODO

* [ ] Multi-master
* [ ] Node removal on destruction, cordoning the node when the
`provisioner` detects that the underlying resource is being destroyed.
* [ ] When a previous `kubeconfig` exists, check
  * if it points to a valid API server
  * if the current token is still valid
  * and try to create a new token otherwise.
* [ ] Support adding new nodes well after the seeder was created
by
  1) loading an existing `kubeconfig`
  2) using the current token if still valid, or creating a new token otherwise
* [ ] The ability to specify the Cloud Provider.
* [ ] Option in the `provisioner` for overriding the node name.
