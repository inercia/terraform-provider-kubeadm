IMAGE := dnd-kubeadm

all: image

.PHONY: image
image:
	@echo "Builing image..."
	cd image && docker build -t $(IMAGE) .
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

clean:
	sudo rm -rf docker_mirror_*

###################################################################
# CI
###################################################################

# entrypoints: ci-setup

# install the requiirements for the CI environment
# this assumes we are running in Travis, in a Ubuntu distro
ci-setup:
	@echo ">>> Making sure Docker is not running..."
	sudo systemctl stop docker || /bin/true

	@echo ">>> Installing a more modern Docker version..."
	curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
	sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu `lsb_release -cs` stable"
	sudo apt-get update
	sudo apt-get -y -o Dpkg::Options::="--force-confnew" install docker-ce

	@echo ">>> Re-enabling the Docker service..."
	sudo systemctl enable --now docker
