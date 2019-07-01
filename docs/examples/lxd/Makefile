MOD_ENV        := GO111MODULE=on GO15VENDOREXPERIMENT=1
GO             := $(MOD_ENV) go
GOPATH         := $(shell go env GOPATH)
GO_NOMOD       := GO111MODULE=off go
GOPATH_FIRST   := $(shell echo ${GOPATH} | cut -f1 -d':')
GOBIN          := $(shell [ -n "${GOBIN}" ] && echo ${GOBIN} || (echo $(GOPATH_FIRST)/bin))

TF_LXD_PROV_URL   := https://github.com/sl1pm4t/terraform-provider-lxd/releases/download/v1.1.3/terraform-provider-lxd_v1.1.3_linux_amd64.zip
TF_LXD_PROV_GOGET := github.com/sl1pm4t/terraform-provider-lxd

TF_SSH_KEY     = ~/.ssh/terraform
PLUGINS_DIR    = $$HOME/.terraform.d/plugins

export TF_LOG=DEBUG
export TF_VAR_private_key=$(TF_SSH_KEY)
export TF_IN_AUTOMATION=1
export GOPATH
export GOBIN

all:
	@echo ">>> This Makefile does not contain targets for users."

$(TF_SSH_KEY):
	@echo ">>> Generating ssh key for Terraform..."
	mkdir -p `dirname $(TF_SSH_KEY)`
	ssh-keygen -t rsa -N "" -f $(TF_SSH_KEY)

$(PLUGINS_DIR):
	mkdir -p $(PLUGINS_DIR)

###################################################################
# CI
###################################################################

ci-install-lxd:
	# LXD version in Xenial is too old (2.0): we must use the snap
	@echo ">>> Installing LXD snap..."
	sudo apt remove -y --purge lxd lxd-client
	sudo snap install lxd
	sudo sh -c 'echo PATH=/snap/bin:${PATH} >> /etc/environment'
	sudo lxd waitready
	sudo lxd init --auto
	sudo usermod -a -G lxd `whoami`

ci-install-tf-lxd-from-srcs: $(PLUGINS_DIR)
	GO111MODULE=on go get -u $(TF_LXD_PROV_GOGET)
	cd $(GOPATH_FIRST)/src/$(TF_LXD_PROV_GOGET)
	GO111MODULE=on make build
	mv -f terraform-provider-lxd* $(PLUGINS_DIR)/terraform-provider-lxd
	chmod 755 $(PLUGINS_DIR)/terraform-provider-lxd

ci-install-tf-lxd-from-zip: $(PLUGINS_DIR)
	cd /tmp	&& \
		rm -f terraform-provider-lxd* && \
		curl -L -O $(TF_LXD_PROV_URL) && \
		unzip -xU terraform-provider-lxd*.zip && \
		rm -f terraform-provider-lxd*.zip && \
		mv -f terraform-provider-lxd* $(PLUGINS_DIR)/terraform-provider-lxd && \
		chmod 755 $(PLUGINS_DIR)/terraform-provider-lxd

ci-install-tf-lxd:
	@echo ">>> Installing the Terraform/LXD provider..."
	[ -x $(PLUGINS_DIR)/terraform-provider-lxd ] || \
		make ci-install-tf-lxd-from-zip
	@echo ">>> Terraform/LXD provider installed."

ci-check-env:
	@echo ">>> Checking things needed in the Terraform script"
	[ -e /dev/mem     ] || exit 1
	[ -d /lib/modules ] || exit 1

# entrypoints: ci-setup

ci-setup: ci-check-env ci-install-tf-lxd ci-install-lxd $(TF_SSH_KEY)

