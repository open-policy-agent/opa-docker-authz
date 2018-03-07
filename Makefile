.PHONY: all build

VERSION := 0.2.1

all: build

build:
	@docker run -it --rm -e VERSION=$(VERSION) -v $(PWD):/go/src/github.com/open-policy-agent/opa-docker-authz golang:1.8 \
		/go/src/github.com/open-policy-agent/opa-docker-authz/build.sh

image: build
	@docker build -t openpolicyagent/opa-docker-authz:$(VERSION) -f Dockerfile .
