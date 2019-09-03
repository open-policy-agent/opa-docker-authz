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

The managed plugin is a special pre-built Docker image, and as such, has no prior knowledge of the user's intended policy. OPA policy is defined using the [Rego language](https://www.openpolicyagent.org/docs/language-reference.html), which for the purposes of the `opa-docker-authz` plugin, is contained within a file. The plugin needs to be made aware of the location of the policy file, during its installation.

 In order to provide user-defined OPA policy, the plugin is configured with a bind mount; `/etc/docker` is mounted at `/opa` inside the plugin's container, which is its working directory. If you define your policy in a file located at the path `/etc/docker/policies/authz.rego`, for example, it will be available to the plugin at `/opa/policies/authz.rego`.

If the plugin is installed without a reference to a Rego policy file, all authorization requests sent to the plugin by the Docker daemon, fail open, and are authorized by the plugin.

The following steps detail how to install the managed plugin.

Download the `opa-docker-authz` plugin from the Docker Hub (depending on how your Docker environment is configured, you may need to execute the following commands using the `sudo` utility), and specify the location of the policy file, using the `opa-args` key, and an appropriate value:

```
$ docker plugin install --alias opa-docker-authz openpolicyagent/opa-docker-authz-v2:0.5 opa-args="-policy-file /opa/policies/authz.rego"
Plugin "openpolicyagent/opa-docker-authz-v2:<VERSION>" is requesting the following privileges:
 - mount: [/etc/docker]
Do you grant the above permissions? [y/N] y
...
Installed plugin openpolicyagent/opa-docker-authz-v2:<VERSION>
```

Check the plugin is installed and enabled:

```
$ docker plugin ls --format 'table {{.ID}}\t{{.Name}}\t{{.Enabled}}'
ID                  NAME                      ENABLED
cab1329e2a5a        opa-docker-authz:latest   true
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

**Legacy Plugin**

If you prefer to use the legacy plugin, it needs to be started as a container, before applying the same configuration to the Docker daemon, as detailed above:

```
$ docker container run -d --restart=always --name opa-docker-authz \
    -v /run/docker/plugins:/run/docker/plugins \
    -v $HOME/opa/policies:/opa \
    openpolicyagent/opa-docker-authz:0.5 -policy-file /opa/authz.rego
```

### Logs

The activity describing the interaction between the Docker daemon and the authorization plugin, and the authorization decisions made by OPA, can be found in the daemon's logs. Their [location](https://docs.docker.com/config/daemon/#read-the-logs) is dependent on the host operating system configuration.

The following is an abbreviated extract from a Docker daemon log, showing the `opa-docker-authz` plugin evaluating an authorization request sent by the Docker daemon, in response to a user issuing the `docker info` command:

```
msg="2018/08/07 14:33:48 Querying OPA policy data.docker.authz.allow. Input: {" plugin=cab132....
msg="  \"AuthMethod\": \"TLS\"," plugin=cab132....
msg="  \"Body\": null," plugin=cab132....
msg="  \"Headers\": {" plugin=cab132....
msg="    \"Accept-Encoding\": \"gzip\"," plugin=cab132....
msg="    \"User-Agent\": \"Docker-Client/18.06.0-ce (linux)\"" plugin=cab132....
msg="  }," plugin=cab132....
msg="  \"Method\": \"GET\"," plugin=cab132....
msg="  \"Path\": \"/_ping\"," plugin=cab132....
msg="  \"User\": \"rackham\"" plugin=cab132....
msg="}" plugin=cab132....
msg="2018/08/07 14:33:48 Returning OPA policy decision: true" plugin=cab132....
msg="2018/08/07 14:33:48 Querying OPA policy data.docker.authz.allow. Input: {" plugin=cab132....
msg="  \"AuthMethod\": \"TLS\"," plugin=cab132....
msg="  \"Body\": null," plugin=cab132....
msg="  \"Headers\": {" plugin=cab132....
msg="    \"Accept-Encoding\": \"gzip\"," plugin=cab132....
msg="    \"User-Agent\": \"Docker-Client/18.06.0-ce (linux)\"" plugin=cab132....
msg="  }," plugin=cab132....
msg="  \"Method\": \"GET\"," plugin=cab132....
msg="  \"Path\": \"/v1.38/info\"," plugin=cab132....
msg="  \"User\": \"rackham\"" plugin=cab132....
msg="}" plugin=cab132....
msg="2018/08/07 14:33:48 Returning OPA policy decision: true" plugin=cab132....
```

### Uninstall

Uninstalling the `opa-docker-authz` plugin is the reverse of installing. First, remove the configuration applied to the Docker daemon, not forgetting to send a `HUP` signal to the daemon's process.

If you're using the legacy plugin, use the `docker container rm -f opa-docker-authz` command to remove the plugin. Otherwise, use the `docker plugin rm -f opa-docker-authz` command to remove the managed plugin.
