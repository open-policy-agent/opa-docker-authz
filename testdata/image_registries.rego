package docker.authz

default allow := false

allow if {
	# Check if the image is allowed
	valid_image
}

allow if {
	valid_image
}

create_container if {
	input.Method == "POST"

	# Use glob to match all versions of the docker api
	glob.match("/**/containers/create", ["/"], input.Path)
}

allowed_registries := ["public.ecr.aws/"]

allowed_images := ["busybox"]

valid_image if {
	create_container
	strings.any_prefix_match(input.Body.Image, allowed_registries)
}

valid_image if {
	create_container
	input.Body.Image in allowed_images
}
