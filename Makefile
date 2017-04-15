all: build

build: providers/terraform-provider-kubeadm provisioners/terraform-provisioner-kubeadm

providers/terraform-provider-kubeadm:
	cd providers && go build -o $$GOBIN/terraform-provider-kubeadm .

provisioners/terraform-provisioner-kubeadm:
	cd provisioners/kubeadm && go generate
	cd provisioners && go build -o $$GOBIN/terraform-provisioner-kubeadm .

clean:
	rm -f */*/generated.go $$GOPATH/bin/terraform-{provider,provisioner}-kubeadm

################################################

# download all the deps defined in vendor.yml
.PHONY: vendor
vendor:
	govend -v --skipTestFiles

# update to the latest version of the dependencies
vendor-update:
	govend -u -v -l --skipTestFiles

################################################

rpm:
	cd osc && osc build