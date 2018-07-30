# opa-docker-authz

This project is used to show how OPA can help policy-enable an existing service.

In this example, we policy-enable the authorization functionality available in the Docker Engine, which is implemented using a plugin architecture. Plugins were introduced in the Docker Engine in 1.10, as a v1 implementation, and further extended in 1.13, as a v2 implementation. Plugins that adhere to the former are often termed [legacy plugins](https://docs.docker.com/engine/extend/legacy_plugins/), whilst the latter are termed [managed plugins](https://docs.docker.com/engine/extend/).

`opa-docker-authz` is an [authorization plugin](https://docs.docker.com/engine/extend/plugins_authorization/) for the Docker Engine, and can be run as a legacy plugin, or as a managed plugin. The managed plugin is the recommended configuration.

## Usage

See the [detailed example](http://www.openpolicyagent.org/docs/docker-authorization.html) to setup a running example of this plugin.

### Build

A makefile is provided for creating different artifacts, each of which requires Docker:

- `make build` - builds the `opa-docker-authz` binary
- `make image` - builds a Docker image for use as a legacy plugin
- `make plugin` - builds a managed plugin

### Install

To make use of the `opa-docker-authz` plugin, [TLS must be enabled](https://docs.docker.com/engine/security/https/), in order for the Docker daemon to authenticate the client user. The client's X.509 certificate subject common name, should be [configured](https://docs.docker.com/engine/extend/plugins_authorization/#default-user-authorization-mechanism) with the user who is the subject of the authorization request.

**Managed Plugin**

The managed plugin is configured with some default policy, which simply allows all Docker client API calls, irrespective of user. The following steps detail how to install the managed plugin.

Download the `opa-docker-authz` plugin from the Docker Hub (depending on how your Docker environment is configured, you may need to execute the following commands using the `sudo` utility):

```
$ docker plugin install openpolicyagent/opa-docker-authz:0.2.2 --alias opa-docker-authz
Plugin "openpolicyagent/opa-docker-authz:0.2.2" is requesting the following privileges:
 - mount: [/etc/docker]
Do you grant the above permissions? [y/N] y
0.2.2: Pulling from openpolicyagent/opa-docker-authz
63ee1bb73b80: Download complete 
Digest: sha256:f76ed8a2fa08d1c144ddf14b9c872590bc5021482602b182b7035255fc8975ab
Status: Downloaded newer image for openpolicyagent/opa-docker-authz:0.2.2
Installed plugin openpolicyagent/opa-docker-authz:0.2.2
```

Check the plugin is installed and enabled:

```
$ docker plugin ls --format 'table {{.ID}}\t{{.Name}}\t{{.Enabled}}'
ID                  NAME                      ENABLED
ef6e3b335fa9        opa-docker-authz:latest   true
```

With the plugin installed and enabled, the Docker daemon needs to be configured to make use of the plugin. There are a couple of ways of doing this, but perhaps the easiest is to add a configuration option to the daemon's configuration file (usually `/etc/docker/daemon.json`):

```json
{
    "authorization-plugins": ["opa-docker-authz"]
}
```

To update the Docker daemon's configuration, send a `HUP` signal to its process:

```
$ sudo kill -HUP $(pidof dockerd)
```

The Docker daemon will now send authorization requests for all Docker client API calls, to the `opa-docker-authz` plugin, for evaluation.

Of course, the default policy is of little use, as it simply allows all requests to be authorized. To enable custom policy use for `opa-docker-authz`, the plugin is configured with a bind mount; `/etc/docker` is mounted at `/opa` inside the plugin's container. If you define your policy in a file located at the path `/etc/docker/policies/authz.rego`, for example, it will be available to the plugin at `/opa/policies/authz.rego`.

To have the `opa-docker-authz` plugin make use of the user-defined policy, specify the additional arguments during plugin installation:

```
$ docker plugin install openpolicyagent/opa-docker-authz:0.2.2 \
    opa-args="-policy-file /opa/policies/authz.rego" \
    --alias opa-docker-authz
```

If an alternate host location is preferred for the bind mount, then it's possible to set the source during plugin installation. For example, if policy files are located in `$HOME/opa/policies`, then a policy file called `authz.rego` can be made available to the plugin, with the following:

```
$ docker plugin install openpolicyagent/opa-docker-authz:0.2.2 \
    policy.source=$HOME/opa/policies \
    opa-args="-policy-file /opa/authz.rego" \
    --alias opa-docker-authz
```

**Legacy Plugin**

If you prefer to use the legacy plugin, it needs to be started as a container, before applying the same configuration to the Docker daemon, as detailed above:

```
$ docker container run -d --restart=always --name opa-docker-authz \
    -v /run/docker/plugins:/run/docker/plugins \
    -v $HOME/opa/policies:/opa \
    openpolicyagent/opa-docker-authz:0.2.2 -policy-file /opa/authz.rego
```

### Uninstall

Uninstalling the `opa-docker-authz` plugin is the reverse of installing. First, remove the configuration applied to the Docker daemon, not forgetting to send a `HUP` signal to the daemon's process.

If you're using the legacy plugin, use the `docker container rm -f opa-docker-authz` command to remove the plugin. Otherwise, use the `docker plugin rm -f opa-docker-authz` to remove the managed plugin.