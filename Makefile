MOD_ENV        := GO111MODULE=on GO15VENDOREXPERIMENT=1
GO             := $(MOD_ENV) go
GOPATH         := $(shell go env GOPATH)
GO_NOMOD       := GO111MODULE=off go
GOPATH_FIRST   := $(shell echo ${GOPATH} | cut -f1 -d':')
GOBIN          := $(shell [ -n "${GOBIN}" ] && echo ${GOBIN} || (echo $(GOPATH_FIRST)/bin))
GO_VERSION     := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_VERSION_MAJ := $(shell echo $(GO_VERSION) | cut -f1 -d'.')
GO_VERSION_MIN := $(shell echo $(GO_VERSION) | cut -f2 -d'.')

TEST           ?= $$(go list ./... |grep -v 'vendor')
GOFMT_FILES    ?= $$(find . -name '*.go' |grep -v vendor)
WEBSITE_REPO   = github.com/hashicorp/terraform-website
WIKI_REPO      = $(shell echo `pwd`.wiki)

TRAVIS_BUILDID := $(shell echo "build-$$RANDOM")
# from https://hub.docker.com/r/travisci/ci-garnet/tags/
TRAVIS_INSTANCE := "travisci/ci-garnet:packer-1515445631-7dfb2e1"

export GOPATH
export GOBIN

export TRAVIS_BUILDID
export TRAVIS_INSTANCE

# for some unknown reason, "provisioners" are only recognized in this directory
PLUGINS_DIR    = $$HOME/.terraform.d/plugins

all: build

default: build

build: fmtcheck build-forced

$(PLUGINS_DIR):
	mkdir -p $(PLUGINS_DIR)

build-forced: $(PLUGINS_DIR)
	$(GO) build -v -o $(PLUGINS_DIR)/terraform-provider-kubeadm     ./cmd/terraform-provider-kubeadm
	$(GO) build -v -o $(PLUGINS_DIR)/terraform-provisioner-kubeadm  ./cmd/terraform-provisioner-kubeadm

generate:
	cd internal/assets && $(GO) generate -x
	cd pkg/provider    && $(GO) generate -x
	cd pkg/provisioner && $(GO) generate -x

install: build-forced

################################################

test: fmtcheck
	$(GO) test $(TEST) || exit 1
	echo $(TEST) | \
		$(MOD_ENV) xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc: fmtcheck
	TF_ACC=1 $(GO) test $(TEST) -v $(TESTARGS) -timeout 120m

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo ">>> ERROR: Set TEST to a specific package. For example,"; \
		echo ">>>  make test-compile TEST=./pkg/provisioner"; \
		exit 1; \
	fi
	$(GO) test -c $(TEST) $(TESTARGS)

################################################

vet:
	@echo ">>> Checking code with 'go vet'"
	@$(GO) vet $$($(GO) list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/utils/gofmtcheck.sh'"

errcheck:
	@sh -c "'$(CURDIR)/utils/errcheck.sh'"


################################################
# CI targets (mainly for Travis)

ci-install-deps:
	# LXD version in Xenial is too old (2.0): we must use the snap
	@echo ">>> Installing LXD snap..."
	sudo apt remove -y --purge lxd lxd-client
	sudo snap install lxd
	sudo sh -c 'echo PATH=/snap/bin:${PATH} >> /etc/environment'
	sudo lxd waitready
	sudo lxd init --auto
	sudo usermod -a -G lxd travis

	mkdir -p "$TRAVIS_BUILD_DIR/snaps-cache"

	@echo ">>> Installing other dependencies, like kubectl..."
	sudo apt-get update && sudo apt-get install -y apt-transport-https
	curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
	echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee -a /etc/apt/sources.list.d/kubernetes.list
	sudo apt-get update
	sudo apt install -y kubectl unzip

	# install Terraform
	@echo ">>> Installing Terraform..."
	wget https://releases.hashicorp.com/terraform/0.11.13/terraform_0.11.13_linux_amd64.zip
	unzip terraform_0.11.13_linux_amd64.zip
	sudo mv terraform /usr/local/bin/

ci-save-env:
	# NOTE: "sudo" in travis resets the environment to "safe" values
	#       (loaded from "/etc/environment"), so we save our current env
	#       in that file
	@env PATH=/snap/bin:${PATH} > /tmp/environment
	@echo ">>> Current environment:"
	@cat /tmp/environment
	@sudo mv -f /tmp/environment /etc/environment

ci-tests: 
	@make build test vet

ci-deploy-lxd:
	@make -C docs/examples/lxd ci

ci-local:
	@echo "Running Travis locally..."
	docker run --name $(TRAVIS_BUILDID) \
		-v ${PWD}:/src \
		-v ${PWD}/utils/travis.sh:/usr/bin/travis.sh \
		-dit $(TRAVIS_INSTANCE) /sbin/init
	docker exec -it $(TRAVIS_BUILDID) /usr/bin/travis.sh

################################################

wiki:
	@echo ">>> Copying markdown file to $(WIKI_REPO)"
	@rsync -av --delete \
		--exclude=.git \
		--exclude=examples \
		docs/ $(WIKI_REPO)/
	@echo ">>> Done. You must commit changes in the wiki repo!"

################################################

rpm:
	cd osc && osc build
