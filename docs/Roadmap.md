# Roadmap and TODO

* [ ] Node removal on destruction, cordoning the node when the
`provisioner` detects that the underlying resource is being destroyed.
* [ ] When a previous `kubeconfig` exists, check if it points to a 
valid API server and try to create a token there.
  * [ ] Support adding new nodes well after the seeder was created
by 1) loading an existing `kubeconfig` 2) creating a new token 3) using that token 
