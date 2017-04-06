all: build

build: providers/terraform-provider-kubeadm provisioners/terraform-provisioner-kubeadm

providers/terraform-provider-kubeadm:
	cd providers && go build -o $$GOPATH/bin/terraform-provider-kubeadm .

provisioners/terraform-provisioner-kubeadm:
	cd provisioners/kubeadm && go generate
	cd provisioners && go build -o $$GOPATH/bin/terraform-provisioner-kubeadm .

clean:
	rm -f */*/generated.go $$GOPATH/bin/terraform-{provider,provisioner}-kubeadm

################################################

.PHONY: vendor
vendor:
	govend -v --skipTestFiles

vendor-update:
	govend -u -v -l --skipTestFiles
