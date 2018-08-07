FROM scratch

LABEL maintainer="Torin Sandall <torinsandall@gmail.com>"

COPY opa-docker-authz /opa-docker-authz

ENTRYPOINT ["/opa-docker-authz"]
