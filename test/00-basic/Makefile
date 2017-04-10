
# CaaSP
CAASP_PREFIX = t00

# common options for ssh
SSH_OPTS=-o "StrictHostKeyChecking no" -o "UserKnownHostsFile /dev/null"
SSH_KEY=../ssh/id_rsa

####################################################################
# test
####################################################################

# create
.PHONY: test-00
test-00-apply:
	terraform apply

.PHONY: test-00-destroy
test-00-destroy:
	-terraform destroy
	-rm -rf terraform.tfstate* kubeconfig ~/.libvirt/images/$(CAASP_PREFIX)_*

test-00: test-00-apply

test-00-ssh-master:
	ssh -i $(SSH_KEY) $(SSH_OPTS) root@`virt-addr master $(CAASP_PREFIX)net`

test-00-ssh-minion0:
	ssh -i $(SSH_KEY) $(SSH_OPTS) root@`virt-addr minion0 $(CAASP_PREFIX)net`
