# Copyright 2016 The OPA Authors.  All rights reserved.
# Use of this source code is governed by an Apache2
# license that can be found in the LICENSE file.

# METADATA
# description: |
#   The "package" provides namespaces the rules contained inside this
#   definition. The package directive controls where the Virtual Documents
#   defined by the rules are located under the Data API. In this case,
#   the documents are available under /v1/data/docker/authz.
package docker.authz

# METADATA
# description: |
#   allow defines a document that is the boolean value true if (and only if) all
#   of the expressions in the body are true. If any of the expressions in the
#   body are false, the document is undefined. Rego allows you to omit the "=
#   true" portion for conciseness.
# entrypoint: true
allow if {
	not invalid_network
	not seccomp_unconfined
	valid_user_role
}

invalid_network if {
	# Example for create operations
	# Use glob.Match to ensure all docker api versions are validated
	glob.match("/**/containers/create", ["/"], input.Path)

	# These expressions assert that a container with a special label must be
	# connected to a specific network.
	labels["com.example/deployment"] == "prod"
	input.Body.HostConfig.NetworkMode != "prod-network"
}

seccomp_unconfined if {
	# Use glob to match all versions of the docker api
	glob.match("/**/containers/create", ["/"], input.Path)

	# This expression asserts that the string on the right hand side exists
	# within the array SecurityOpt referenced on the left hand side.
	"seccomp=unconfined" in input.Body.HostConfig.SecurityOpt
}

# valid_user_role defines a document that is the boolean value true if this is
# a write request and the user is allowed to perform writes.
valid_user_role if {
	input.Method != "GET"
	user.readOnly == false
}

# valid_user_role is defined again here to handle read requests. When a rule
# like this is defined multiple times, the rule definition must ensure that
# only one instance evaluates successfully in a given query. If multiple
# instances evaluated successfully, it indicates a conflict.
valid_user_role if {
	input.Method == "GET"
	user
}

# labels defines an object document that simply contains the labels from the
# requested container.
labels[key] := value if {
	some key, value in input.Body.Labels
}

# Create a shorthand rule for user mapping from input
user := users[input.Headers["Authz-User"]]

# Example users. A real implementation would likely have users provided
# as data rather than coded directly into the policy.
users := {
	"bob": {"readOnly": true},
	"alice": {"readOnly": false},
}
