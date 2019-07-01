IMAGE := dnd-kubeadm

all: image

image:
	@echo "Builing image..."
	docker build -t $(IMAGE) .
	@echo "done!"

# target for testing the image
# once the image is running, you can "ssh -p 8022 root@127.0.0.1"
# with password "linux"
run:
	docker run --rm --privileged=true \
		-v $$HOME/.ssh/id_rsa.pub:/root/.ssh/authorized_keys:ro \
		-v /sys/fs/cgroup:/sys/fs/cgroup:ro \
		-p 8022:22 \
		--name $(IMAGE) \
		$(IMAGE)

###################################################################
# CI
###################################################################

ci-setup:
	@echo "No setup is necessary"

ci: ci-setup
	@echo ">>> Deploying in Docker"
	terraform --version
	terraform init
	terraform apply -auto-approve
	terraform output
	terraform destroy -auto-approve
	# TODO: do a `kubectl get nodes`
