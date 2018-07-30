FROM alpine AS default-policy

RUN echo -e 'package docker.authz\n\
allow = true'\
>> /default.rego

FROM scratch AS image

LABEL maintainer="Torin Sandall <torinsandall@gmail.com>"

COPY opa-docker-authz /opa-docker-authz

ENTRYPOINT ["/opa-docker-authz"]

FROM image

COPY --from=default-policy /default.rego /default.rego

