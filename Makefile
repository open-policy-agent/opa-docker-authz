.PHONY: all build

VERSION ?= 0.9
GO_VERSION := 1.24.3
GOLANGCI_LINT_VERSION := v2.1.6
REPO ?= openpolicyagent/opa-docker-authz-v2

REGAL_FORMAT ?= pretty
ifeq (true,$(GITHUB_ACTIONS))
REGAL_FORMAT = github
endif

all: build

check:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:${GOLANGCI_LINT_VERSION} golangci-lint run -v

	regal lint --format $(REGAL_FORMAT) .

fmt:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:${GOLANGCI_LINT_VERSION} golangci-lint run -v --fix

test:
	go test

build:
	@docker container run --rm \
		-e VERSION=$(VERSION) \
		-v $(PWD):/go/src/github.com/open-policy-agent/opa-docker-authz \
		-w /go/src/github.com/open-policy-agent/opa-docker-authz \
		golang:$(GO_VERSION) \
		./build.sh

image: build
	@docker image build \
		--tag $(REPO):$(VERSION) \
		.

plugin: build
	VERSION=$(VERSION) REPO=$(REPO) ./plugin.sh

plugin-push:
	@for plugin in `docker plugin ls --format '{{.Name}}'`; do \
		if [ "$$plugin" = "$(REPO)-v2:$(VERSION)" ]; then \
		    echo "\nPushing plugin $(REPO)-v2:$(VERSION) ..."; \
            docker plugin push $(REPO)-v2:$(VERSION); \
			exit; \
		fi \
	done; \
	echo "\nNo local copy of $(REPO)-v2:$(VERSION) exists, create it before attempting push"

clean:
	@if [ -f ./opa-docker-authz ]; then \
		echo "\nRemoving opa-docker-authz binary ..."; \
		rm -rvf ./opa-docker-authz; \
	fi
	@for plugin in `docker plugin ls --format '{{.Name}}'`; do \
		if [ "$$plugin" = "$(REPO)-v2:$(VERSION)" ]; then \
		    echo "\nRemoving local copy of plugin $(REPO):$(VERSION) ..."; \
            docker plugin rm -f $(REPO)-v2:$(VERSION); \
		fi \
	done
