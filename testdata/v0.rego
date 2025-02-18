# Copyright 2016 The OPA Authors.  All rights reserved.
# Use of this source code is governed by an Apache2
# license that can be found in the LICENSE file.

# The "package" provides namespaces the rules contained inside this
# definition. The package directive controls where the Virtual Documents
# defined by the rules are located under the Data API. In this case,
# the documents are available under /v1/data/docker/authz.
package docker.authz

# allow defines a document that is the boolean value true if (and only if) all
# of the expressions in the body are true. If any of the expressions in the
# body are false, the document is undefined. Rego allows you to omit the "=
# true" portion for conciseness.
allow {
	not invalid_network
	not seccomp_unconfined
	valid_user_role
}

invalid_network {
	# These expressions assert that a container with a special label must be
	# connected to a specific network.
	labels["com.example/deployment"] = "prod"
	input.Path = "/v1.23/containers/create"
	input.Body.HostConfig.NetworkMode != "prod-network"
}
