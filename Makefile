.PHONY: all build

VERSION := 0.2.2
OPA_VERSION := 0.8.0
GO_VERSION := 1.10
REPO := openpolicyagent/opa-docker-authz

all: build

build:
	@docker container run --rm \
		-e VERSION=$(VERSION) \
		-e OPA_VERSION=$(OPA_VERSION) \
		-v $(PWD):/go/src/github.com/open-policy-agent/opa-docker-authz \
		-w /go/src/github.com/open-policy-agent/opa-docker-authz \
		golang:$(GO_VERSION) \
		./build.sh
	@sudo rm -rf ./vendor

image: build
	@docker image build \
		--target image \
		--tag $(REPO):$(VERSION) \
		.
 
plugin: build
	@mkdir ./rootfs
	@echo "\nCreating root filesystem for plugin ..."
	@docker image build -t rootfsimage .
	@id=$$(docker container create rootfsimage true) && \
	sudo docker container export $$id | sudo tar -x -C ./rootfs && \
	docker container rm -f $$id
	@docker image rm -f rootfsimage
	@echo "\nCreating plugin $(REPO):$(VERSION) ..."
	@sudo docker plugin create $(REPO):$(VERSION) .
	@sudo rm -rf ./rootfs

plugin-push: plugin
	@echo "\nPushing plugin $(REPO):$(VERSION) ..."
	@docker plugin push $(REPO):$(VERSION)

clean:
	@echo "\nRemoving opa-docker-authz binary ..."
	@rm -rfv ./opa-docker-authz
	@echo "\nRemoving local copy of plugin $(REPO):$(VERSION) ..."
	@docker plugin rm -f $(REPO):$(VERSION)
