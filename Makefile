GO             := GO111MODULE=on GO15VENDOREXPERIMENT=1 go
GO_NOMOD       := GO111MODULE=off go
GOPATH_FIRST   := $(shell echo ${GOPATH} | cut -f1 -d':')
GO_BIN         := $(shell [ -n "${GOBIN}" ] && echo ${GOBIN} || (echo $(GOPATH_FIRST)/bin))
GO_VERSION     := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_VERSION_MAJ := $(shell echo $(GO_VERSION) | cut -f1 -d'.')
GO_VERSION_MIN := $(shell echo $(GO_VERSION) | cut -f2 -d'.')

all: build

build: providers/terraform-provider-kubeadm provisioners/terraform-provisioner-kubeadm

providers/terraform-provider-kubeadm:
	@echo ">>> Building the provider..."
	cd providers && $(GO) build -o $(GOBIN)/terraform-provider-kubeadm .

provisioners/terraform-provisioner-kubeadm:
	@echo ">>> Generating code in the provisioner..."
	cd provisioners/kubeadm && $(GO) generate
	@echo ">>> Building the provisioner..."
	cd provisioners && $(GO) build -o $(GOBIN)/terraform-provisioner-kubeadm .

clean:
	rm -f */*/generated.go $$GOPATH/bin/terraform-{provider,provisioner}-kubeadm

################################################

rpm:
	cd osc && osc build
