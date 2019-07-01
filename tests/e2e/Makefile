all: test

setup:
	[ -n "$$E2E_ENV" ] && make -C "$$E2E_ENV" ci-setup

test: setup
	@echo "Running tests suite:"
	@echo
	@for i in [0-9]*-*.sh ; do sh $$i ; done
