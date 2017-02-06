.PHONY: all build

all: build

build:
	@docker run -it --rm -v $(PWD):/go/src/github.com/open-policy-agent/opa-docker-authz golang:1.6 \
		/go/src/github.com/open-policy-agent/opa-docker-authz/build.sh
