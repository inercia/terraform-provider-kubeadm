GO             := GO111MODULE=on GO15VENDOREXPERIMENT=1 go
GO_NOMOD       := GO111MODULE=off go
GOPATH_FIRST   := $(shell echo ${GOPATH} | cut -f1 -d':')
GO_BIN         := $(shell [ -n "${GOBIN}" ] && echo ${GOBIN} || (echo $(GOPATH_FIRST)/bin))
GO_VERSION     := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_VERSION_MAJ := $(shell echo $(GO_VERSION) | cut -f1 -d'.')
GO_VERSION_MIN := $(shell echo $(GO_VERSION) | cut -f2 -d'.')
PKG_NAME       = kubeadm

TEST           ?= $$(go list ./... |grep -v 'vendor')
GOFMT_FILES    ?= $$(find . -name '*.go' |grep -v vendor)
WEBSITE_REPO   = github.com/hashicorp/terraform-website
WIKI_REPO      = $(shell echo `pwd`.wiki)

# for some unknown reason, "provisioners" are only recognized in this directory
PLUGINS_DIR    = $$HOME/.terraform.d/plugins

all: build

default: build

build: fmtcheck build-forced

build-forced:
	mkdir -p $(PLUGINS_DIR)
	$(GO) build -o $(PLUGINS_DIR)//terraform-provider-kubeadm .
	cp -f $(PLUGINS_DIR)/terraform-provider-kubeadm \
	      $(PLUGINS_DIR)/terraform-provisioner-kubeadm

generate:
	cd kubeadm && $(GO) generate -x

################################################

test: fmtcheck
	$(GO) test $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 $(GO) test $(TESTARGS) -timeout=30s -parallel=4

testacc: fmtcheck
	TF_ACC=1 $(GO) test $(TEST) -v $(TESTARGS) -timeout 120m

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	$(GO) test -c $(TEST) $(TESTARGS)

################################################

vet:
	@echo "go vet ."
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

wiki:
	@echo "Copying markdown file to $(WIKI_REPO)"
	@rsync -av --delete \
		--exclude=.git \
		--exclude=examples \
		docs/ $(WIKI_REPO)/
	@echo "Done. You must commit changes in the wiki repo!"

################################################

rpm:
	cd osc && osc build
