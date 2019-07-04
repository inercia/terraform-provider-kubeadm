all: ci-tests

ci-setup:
	mkdir -p "$TRAVIS_BUILD_DIR/snaps-cache"

	@echo ">>> Installation stuff specific to the $$E2E_ENV environment"
	[ -n "$$E2E_ENV" ] && make -C "$$E2E_ENV" ci-setup

	@echo ">>> Installing other dependencies, like kubectl, terraform..."
	sudo apt-get update && sudo apt-get install -y apt-transport-https
	curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
	echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee -a /etc/apt/sources.list.d/kubernetes.list
	sudo apt-get update
	sudo apt install -y kubectl unzip

	@echo ">>> Installing Terraform..."
	wget https://releases.hashicorp.com/terraform/0.11.13/terraform_0.11.13_linux_amd64.zip
	unzip terraform_0.11.13_linux_amd64.zip
	sudo mv terraform /usr/local/bin/

ci-tests:
	@echo "Running tests suite:"
	@echo
	@for d in `pwd`/[0-9]* ; do \
		echo "Running testsuite $$d" ; \
		for i in $$d/[0-9]*-*.sh ; do \
			echo "Running $$i" ; \
			sh $$i ; \
		done ; \
	done

