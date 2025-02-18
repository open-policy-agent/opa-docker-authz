package docker.authz

default allow := false

allow if {
	input.Method == "GET"
}
