FROM alpine

MAINTAINER Torin Sandall <torinsandall@gmail.com>

ADD opa-docker-authz /opa-docker-authz

ENTRYPOINT ["/opa-docker-authz"]
