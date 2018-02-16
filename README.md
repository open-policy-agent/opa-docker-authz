# opa-docker-authz

This project is used to show how OPA can help policy-enable an existing service.

In this example, we policy-enable the authorization functionality available in
Docker 1.10 and later.

## Usage

See the [detailed example](http://www.openpolicyagent.org/docs/docker-authorization.html) to setup a running example of this plugin.

### Build

To build the plugin run `make`. The build requires Docker.

### Install

The plugin can be started with no options. It may require sudo depending on your
machine's Docker configuration permissions:

    $ opa-docker-authz

- By default, the plugin will listen for requests (from Docker) on :8080 and
  read an OPA policy out of `policy.rego`. See `-h` for options.

The following command line argument enables the authorization plugin within Docker:

    --authorization-plugin=opa-docker-authz

On Ubuntu 16.04 this is done by overriding systemd configuration (requires root):

    $ sudo mkdir -p /etc/systemd/system/docker.service.d
    $ sudo tee -a /etc/systemd/system/docker.service.d/override.conf > /dev/null <<EOF
    [Service]
    ExecStart=
    ExecStart=/usr/bin/docker daemon -H fd:// --authorization-plugin=opa-docker-authz
    EOF
    $ sudo systemctl daemon-reload
    $ sudo service docker restart
