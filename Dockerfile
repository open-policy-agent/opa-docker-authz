FROM --platform=$BUILDPLATFORM alpine:latest AS certs
RUN apk --update add ca-certificates

FROM scratch
ARG TARGETOS
ARG TARGETARCH

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY opa-docker-authz-${TARGETOS}-${TARGETARCH} /opa-docker-authz

ENTRYPOINT ["/opa-docker-authz"]
