FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch

LABEL maintainer="Torin Sandall <torinsandall@gmail.com>"

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY opa-docker-authz /opa-docker-authz

ENTRYPOINT ["/opa-docker-authz"]
