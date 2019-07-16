
###################################################################
# CI
###################################################################

# entrypoints: ci-setup, ci-cleanup

ci-setup:
	@echo ">>> No setup to do for AWS..."

ci-cleanup:
	terraform init
	terraform destroy --auto-approve
	rm -f *.log
