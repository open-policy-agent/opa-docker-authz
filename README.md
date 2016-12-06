# opa-docker-authz

This project is used to show how OPA can help policy-enable an existing service.

In this example, we policy-enable the authorization functionality available in Docker 1.10 and later.

## Usage

See the [detailed example](http://www.openpolicyagent.org/examples/docker-authorization/) to setup a running example of this plugin.

### Build

To build the plugin, just run:

    $ go get ./...
    $ go build -o opa-docker-authz

This assumes you are running on Linux and have Go 1.6 or later on your machine. You must have $GOPATH set.

If you are running on OS X and want to cross compile for Linux, you can do so as follows:

    $ docker run -it --rm -v $PWD:/go/src/github.com/open-policy-agent/opa-docker-authz golang:1.6 bash
    $ cd /go/src/github.com/open-policy-agent/opa-docker-authz/
    $ go get ./...
    $ go build -o opa-docker-authz
    $ exit

### Install

The plugin can be started with no options. It may require sudo depending on your machine's Docker configuration permissions:

    $ opa-docker-authz

- By default, the plugin will listen for requests (from Docker) on :8080 and contacts OPA on :8181.

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

### Testing

The plugin will upsert a policy definition (by default, "example.rego") into OPA on startup and then establish a file watch to be notified when the definition changes. Each time the definition changes, the plugin will upsert into OPA.
