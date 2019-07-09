MOD_ENV        := GO111MODULE=on GO15VENDOREXPERIMENT=1
GO             := $(MOD_ENV) go
GOPATH         := $(shell go env GOPATH)
GO_NOMOD       := GO111MODULE=off go
GOPATH_FIRST   := $(shell echo ${GOPATH} | cut -f1 -d':')
GOBIN          := $(shell [ -n "${GOBIN}" ] && echo ${GOBIN} || (echo $(GOPATH_FIRST)/bin))
GO_VERSION     := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_VERSION_MAJ := $(shell echo $(GO_VERSION) | cut -f1 -d'.')
GO_VERSION_MIN := $(shell echo $(GO_VERSION) | cut -f2 -d'.')

# directories with sources
SRC_DIRS        = pkg internal

TEST           ?= $$(go list ./... 2>/dev/null |grep -v 'vendor')
GOFMT_FILES    ?= $$(find $(SRC_DIRS) -name '*.go' |grep -v vendor)
WEBSITE_REPO   = github.com/hashicorp/terraform-website
WIKI_REPO      = $(shell echo `pwd`.wiki)

# for some unknown reason, "provisioners" are only recognized in this directory
PLUGINS_DIR    = $$HOME/.terraform.d/plugins

# the deployment used for running the E2E tests
E2E_ENV         := $(shell echo `pwd`)/docs/examples/dnd

export GOPATH
export GOBIN
export E2E_ENV


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

.PHONY: vendor
vendor:
	$(GO) mod tidy
	$(GO) mod vendor

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

tests-e2e: ci-tests-e2e

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
# CI targets (for Travis)

ci-save-env:
	# NOTE: "sudo" in travis resets the environment to "safe" values
	#       (loaded from "/etc/environment"), so we save our current env
	#       in that file
	@env PATH=/snap/bin:${PATH} > /tmp/environment
	@echo ">>> Current environment:"
	@cat /tmp/environment
	@sudo mv -f /tmp/environment /etc/environment

ci-tests-style: fmtcheck vet errcheck

ci-tests-unit: test

ci-tests-e2e: build
	@make -C tests/e2e ci-tests

ci: ci-tests

# entrypoints: ci-tests and ci-setup

ci-tests: ci-tests-unit ci-tests-style ci-tests-e2e

ci-setup:
	@make -C tests/e2e ci-setup
	@make              ci-save-env

################################################

wiki:
	@echo ">>> Copying markdown file to $(WIKI_REPO)"
	@rm -rf $(WIKI_REPO)/*
	@rsync -av --delete \
		--exclude=.git \
		--exclude=examples \
		docs/ $(WIKI_REPO)/
	@echo ">>> Done. You must commit changes in the wiki repo!"

################################################

rpm:
	cd osc && osc build
