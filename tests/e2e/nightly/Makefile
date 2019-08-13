
# see https://github.com/heptio/sonobuoy

ci-setup:
	@echo ">>> Installing additional stuff for the nightly build..."
	go get -u -v github.com/heptio/sonobuoy

ci-tests:
	@echo ">>> Running nightly job: sonobuoy tests..."
	KUBECONFIG=$E2E_ENV/kubeconfig.local sonobuoy run --wait


