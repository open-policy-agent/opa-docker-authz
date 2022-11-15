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

The managed plugin is a special pre-built Docker image, and as such, has no prior knowledge of the user's intended policy. OPA policy is defined using the [Rego language](https://www.openpolicyagent.org/docs/language-reference.html), which for the purposes of the `opa-docker-authz` plugin, is either contained within a file (using the `-policy-file` argument) or fetched from bundles through an OPA [configuration](https://www.openpolicyagent.org/docs/latest/configuration/) file (using the `-config-file` argument). Since the latter option allows not just remote bundles, but any of the OPA management features such as decision logging, it is the recommended choice. The plugin needs to be made aware of either the location of the policy file, or the config file, during its installation.

In order to provide user-defined OPA policy or config, the plugin is configured with a bind mount; `/etc/docker` is mounted at `/opa` inside the plugin's container, which is its working directory. If you define your config in a file located at the path `/etc/docker/config/opa-conf.yaml`, for example, it will be available to the plugin at `/opa/config/opa-conf.yaml`.

If the plugin is installed without a reference to a Rego policy file, or a config file, all authorization requests sent to the plugin by the Docker daemon, fail open, and are authorized by the plugin.

The following steps detail how to install the managed plugin.

Download the `opa-docker-authz` plugin from the Docker Hub (depending on how your Docker environment is configured, you may need to execute the following commands using the `sudo` utility), and specify the location of the policy file, or config file, using the `opa-args` key, and an appropriate value:

```
$ docker plugin install --alias opa-docker-authz openpolicyagent/opa-docker-authz-v2:0.8 opa-args="-config-file /opa/config/opa-conf.yaml"
Plugin "openpolicyagent/opa-docker-authz-v2:<VERSION>" is requesting the following privileges:
 - mount: [/etc/docker]
Do you grant the above permissions? [y/N] y
...
Installed plugin openpolicyagent/opa-docker-authz-v2:<VERSION>
```

Check the plugin is installed and enabled:

```
$ docker plugin ls
ID                  NAME                      ENABLED
cab1329e2a5a        opa-docker-authz:latest   true
```

With the plugin installed and enabled, the Docker daemon needs to be configured to make use of the plugin. There are a couple of ways of doing this, but perhaps the easiest is to add a configuration option to the daemon's configuration file (usually `/etc/docker/daemon.json`):

```json
{
    "authorization-plugins": ["openpolicyagent/opa-docker-authz-v2:0.8"]
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
    openpolicyagent/opa-docker-authz:0.6 -policy-file /opa/authz.rego
```

### Logs

If using the plugin with the `-config-file` option, full decision logging capabilities - including configuring remote endpoints - is at your disposal.

If using a policy file, the activity describing the interaction between the Docker daemon and the authorization plugin, and the authorization decisions made by OPA, can be found in the daemon's logs. Their [location](https://docs.docker.com/config/daemon/#read-the-logs) is dependent on the host operating system configuration.

Logs are generated in a json format similar to [decision logs](https://www.openpolicyagent.org/docs/latest/management/#decision-logs):

```
{
  "config_hash": "a2e84e38eafd14a816194357860b253becbc739e601cf4307078413a0a578a89",
  "decision_id": "8d4c6d08-b56e-4625-b66c-3e6c00d7a6e7",
  "input": {
    "AuthMethod": "",
    "BindMounts": [],
    "Body": null,
    "Headers": {
      "Content-Length": "0",
      "Content-Type": "text/plain",
      "User-Agent": "Docker-Client/19.03.11 (linux)"
    },
    "Method": "POST",
    "Path": "/v1.40/images/create?fromImage=registry.company.com%3A8885%2Fbash\\u0026tag=latest",
    "PathArr": [
      "",
      "v1.40",
      "images",
      "create"
    ],
    "PathPlain": "/v1.40/images/create",
    "Query": {
      "fromImage": [
        "registry.company.com:8885/bash"
      ],
      "tag": [
        "latest"
      ]
    },
    "User": ""
  },
  "labels": {
    "app": "opa-docker-authz",
    "id": "396f1138-ea63-4be0-9ce0-3184cb20b1dd",
    "opa_version": "v0.18.0",
    "plugin_version": "0.8"
  },
  "result": true,
  "timestamp": "2020-06-16T16:44:54.328705305Z"
}
```

### Input Processing

The Rego `input` document is largely identical to the JSON data structure given to opa-docker-authz by Docker, with the following additions
to enrich the document with additional information and assist policy authoring:
 - PathPlain - the Path portion of the RequestURI (exposed as 'Path'), i.e. without the query string 
 - PathArr - PathPlain split into an array of path elements by '/'
 - BindMounts - an array of bind mount objects, as specified via either 'Binds' or 'Mounts' (see below)
 
#### BindMounts

The BindMounts array is populated with information about the source, readonly status and resolved symlink path of each bind.  The each object in the array
has the schema

```
{
  "Source": "<source path>",
  "ReadOnly": true|false,
  "Resolved": "<resolved source path>"
}
```

where 'Resolved' is either the empty string ("") or the full host path that corresponds to `Source` after resolving any symbolic links. 
This allows for effective policy checking of bind mount sources, including where the true source path is obfuscated with symlinks. This
mitigates against a known trivial bypass of policy that check for binds, for example

```
cd /home/user
ln -sf / root
docker run --rm -it -v/home/user/root:/mnt image
# /mnt is now / in the hostfs
docker run --rm -it -v/home/user/root/var:/mnt image
# /mnt is now /var on the host
```

In each of the above examples, the 'Resolved' path allows for the situation to be detected by policy (it will resolve to "/" and "/var", respectively).

**Note**: in order for the bind mount resolution to work, the opa-docker-authz plugin must have read access to all parts of the filesystem for which
these checks are required by the policy.  The easiest way to achieve this is to run the plugin as a legacy plugin as `root`.  If using a managed plugin,
the `config.json` would need to rebuilt with a custom bind configuration that exposes the relevant parts of the hostfs to the plugin as read only binds. 

### Uninstall

Uninstalling the `opa-docker-authz` plugin is the reverse of installing. First, remove the configuration applied to the Docker daemon, not forgetting to send a `HUP` signal to the daemon's process.

If you're using the legacy plugin, use the `docker container rm -f opa-docker-authz` command to remove the plugin. Otherwise, use the `docker plugin rm -f opa-docker-authz` command to remove the managed plugin.
