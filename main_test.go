package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/docker/go-plugins-helpers/authorization"
)

func TestNormalizeAllowPath(t *testing.T) {
	tests := []struct {
		input    string
		useConf  bool
		expected string
	}{
		{
			input:    "data.policy.rule",
			useConf:  true,
			expected: "/policy/rule",
		},
		{
			input:    "data.policy.rule",
			useConf:  false,
			expected: "data.policy.rule",
		},
		{
			input:    "/policy/rule",
			useConf:  true,
			expected: "/policy/rule",
		},
		{
			input:    "/policy/rule",
			useConf:  false,
			expected: "data.policy.rule",
		},
		{
			input:    "",
			useConf:  true,
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run("Normalize allowPath", func(t *testing.T) {
			result := normalizeAllowPath(tc.input, tc.useConf)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestListBindMounts(t *testing.T) {
	dotDotPath := fmt.Sprintf("%s/../../../../", t.TempDir())
	symlinkSourcePath := t.TempDir()
	symlinkTargetPath := fmt.Sprintf("%s/target", t.TempDir())
	err := os.Symlink(symlinkSourcePath, symlinkTargetPath)

	if err != nil {
		t.Fatalf("Failed to symlink '%s' to '%s' - got %v", symlinkSourcePath, symlinkTargetPath, err)
	}

	tests := []struct {
		statement string
		input     string
		expected  []BindMount
	}{
		{
			statement: "parse a simple bind list",
			input:     `{ "HostConfig": { "Binds" : [ "/var:/home", "volume:/var/lib/app:ro" ] } }`,
			expected:  []BindMount{{"/var", false, "/var"}},
		},
		{
			statement: "expand ..",
			input:     fmt.Sprintf(`{ "HostConfig": { "Binds" : [ "%s:/host" ] } }`, dotDotPath),
			expected:  []BindMount{{dotDotPath, false, "/"}},
		},
		{
			statement: "resolve symlinks",
			input:     fmt.Sprintf(`{ "HostConfig": { "Binds" : [ "%s:/host" ] } }`, symlinkTargetPath),
			expected:  []BindMount{{symlinkTargetPath, false, symlinkSourcePath}},
		},
		{
			statement: "resolve symlinks followed by non-existent directories",
			input:     fmt.Sprintf(`{ "HostConfig": { "Binds" : [ "%s/nonexistent:/host" ] } }`, symlinkTargetPath),
			expected:  []BindMount{{symlinkTargetPath + "/nonexistent", false, symlinkSourcePath + "/nonexistent"}},
		},
		{
			statement: "parse the readonly attribute",
			input:     `{ "HostConfig": { "Binds" : [ "/var:/home:ro", "/var/lib:/mnt:rw" ] } }`,
			expected:  []BindMount{{"/var", true, "/var"}, {"/var/lib", false, "/var/lib"}},
		},
		{
			statement: "handle when neither bind nor mounts provided",
			input:     `{ "HostConfig": {} }`,
			expected:  []BindMount{},
		},
		{
			statement: "handle an invalid binds list",
			input:     `{ "HostConfig": { "Binds" : null } }`,
			expected:  []BindMount{},
		},
		{
			statement: "handle an empty binds list",
			input:     `{ "HostConfig": { "Binds" : [] } }`,
			expected:  []BindMount{},
		},
		{
			statement: "parse a mount list",
			input: `{ "HostConfig": { "Mounts" : [ 
				{ "Source": "/var", "Target": "/mnt", "Type": "bind" },
				{ "Source": "vol", "Target": "/vol", "Type": "volume", "Labels":{"color":"red"} }
				] } }`,
			expected: []BindMount{{"/var", false, "/var"}},
		},
		{
			statement: "parse a readonly mount list",
			input: `{ "HostConfig": { "Mounts" : [ 
				{ "Source": "/var", "Target": "/mnt", "Type": "bind", "ReadOnly": true },
				{ "Source": "/home", "Target": "/home", "Type": "bind" }
				] } }`,
			expected: []BindMount{{"/var", true, "/var"}, {"/home", false, "/home"}},
		},
		{
			statement: "ignore an invalid mount list",
			input: `{ "HostConfig": { "Mounts" : [ 
				{ "Source": "/var", "Target": "/mnt", "Type": "bind", "ReadOnly": true },
				{ "Source1": "/home", "Target": "/home", "Type": "bind" }
				] } }`,
			expected: []BindMount{{"/var", true, "/var"}},
		},
		{
			statement: "ignore a mount list of the wrong type, whlile reading binds",
			input: `{ "HostConfig": { "Binds": ["/var:/mnt/var:ro","/home:/home"],
				"Mounts" : null } }`,
			expected: []BindMount{{"/var", true, "/var"}, {"/home", false, "/home"}},
		},
	}

	for _, tc := range tests {
		t.Run("listBindMounts should "+tc.statement, func(t *testing.T) {
			var body map[string]interface{}
			err := json.Unmarshal([]byte(tc.input), &body)
			if err != nil {
				t.Fatalf("Improper JSON input - got %v for '%s'", err, tc.input)
			}

			result := listBindMounts(body)
			if len(result) > 0 && len(tc.expected) > 0 && !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	// resolve the temp dir itself, as it may contain symlinks (e.g. on macOS)
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to resolve temp dir - got %v", err)
	}

	target := filepath.Join(base, "target")
	file := filepath.Join(base, "file")
	for _, dir := range []string{target, filepath.Join(base, "dir")} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("Failed to create directory '%s' - got %v", dir, err)
		}
	}
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatalf("Failed to create file '%s' - got %v", file, err)
	}

	symlinks := map[string]string{
		"link":    target,
		"dir/rel": "../target",
		"chain":   filepath.Join(base, "link"),
		"loop":    filepath.Join(base, "loop"),
	}
	for name, dest := range symlinks {
		if err := os.Symlink(dest, filepath.Join(base, name)); err != nil {
			t.Fatalf("Failed to symlink '%s' to '%s' - got %v", name, dest, err)
		}
	}

	tests := []struct {
		statement     string
		input         string
		expected      string
		expectedError bool
	}{
		{
			statement: "keep the root path",
			input:     "/",
			expected:  "/",
		},
		{
			statement: "keep a path without symlinks",
			input:     target,
			expected:  target,
		},
		{
			statement: "resolve an absolute symlink",
			input:     filepath.Join(base, "link"),
			expected:  target,
		},
		{
			statement: "resolve a relative symlink",
			input:     filepath.Join(base, "dir/rel"),
			expected:  target,
		},
		{
			statement: "resolve a chain of symlinks",
			input:     filepath.Join(base, "chain"),
			expected:  target,
		},
		{
			statement: "resolve a symlink followed by non-existent directories",
			input:     filepath.Join(base, "link") + "/nonexistent/subdir",
			expected:  target + "/nonexistent/subdir",
		},
		{
			statement: "resolve a relative symlink followed by a non-existent directory",
			input:     filepath.Join(base, "dir/rel") + "/nonexistent",
			expected:  target + "/nonexistent",
		},
		{
			statement: "clean .. before resolving symlinks, like Docker does",
			input:     base + "/nonexistent/../link/../link/nonexistent",
			expected:  target + "/nonexistent",
		},
		{
			statement: "remove trailing slashes",
			input:     filepath.Join(base, "link") + "/nonexistent/",
			expected:  target + "/nonexistent",
		},
		{
			statement:     "reject a relative path",
			input:         "relative/path",
			expectedError: true,
		},
		{
			statement:     "reject a symlink loop",
			input:         filepath.Join(base, "loop") + "/nonexistent",
			expectedError: true,
		},
		{
			statement:     "reject a path below a regular file",
			input:         file + "/nonexistent",
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run("resolvePath should "+tc.statement, func(t *testing.T) {
			result, err := resolvePath(tc.input)
			if (err != nil) != tc.expectedError {
				t.Fatalf("Expected error: %v, got: %v", tc.expectedError, err)
			}
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name           string
		policyFile     string
		allowPath      string
		request        authorization.Request
		expectedResult bool
		expectedError  bool
		skipPing       bool
	}{
		{
			name:       "PING",
			policyFile: "testdata/default_allow.rego",
			allowPath:  "data.docker.authz.allow",
			request: authorization.Request{
				RequestMethod: "HEAD",
				RequestURI:    "/_ping",
			},
			expectedResult: true,
			expectedError:  false,
			skipPing:       true,
		},
		{
			name:       "Policy file OK with GET method",
			policyFile: "testdata/default_allow.rego",
			allowPath:  "data.docker.authz.allow",
			request: authorization.Request{
				RequestMethod: "GET",
				RequestHeaders: map[string]string{
					"Authz-User": "alice",
				},
				RequestURI: "/v1.23/containers/json",
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:       "Container create allowed for user alice",
			policyFile: "example.rego",
			allowPath:  "data.docker.authz.allow",
			request: authorization.Request{
				RequestMethod: "POST",
				RequestURI:    "/v1.47/containers/create",
				RequestBody:   []byte(`{"Image": "busybox"}`),
				RequestHeaders: map[string]string{
					"Content-Type": "application/json",
					"Authz-User":   "alice",
				},
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:       "Container create not allowed for readonly user bob",
			policyFile: "example.rego",
			allowPath:  "data.docker.authz.allow",
			request: authorization.Request{
				RequestMethod: "POST",
				RequestURI:    "/v1.23/containers/create",
				RequestBody:   []byte(`{"Image": "busybox"}`),
				RequestHeaders: map[string]string{
					"Content-Type": "application/json",
					"Authz-User":   "bob",
				},
			},
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:           "Policy file OK with non-GET method",
			policyFile:     "example.rego",
			allowPath:      "data.docker.authz.allow",
			request:        authorization.Request{RequestMethod: "GET"},
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:           "Policy file does not exist",
			policyFile:     "nonexistent.rego",
			allowPath:      "data.docker.authz.allow",
			request:        authorization.Request{RequestMethod: "GET"},
			expectedResult: true, // policy file nonexistent.rego does not exist, failing open and allowing request
			expectedError:  true,
		},
		{
			name:           "Test v0 policy file",
			policyFile:     "testdata/v0.rego",
			allowPath:      "data.docker.authz.allow",
			request:        authorization.Request{RequestMethod: "GET"},
			expectedResult: false,
			expectedError:  true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plugin := DockerAuthZPlugin{
				policyFile: tc.policyFile,
				allowPath:  tc.allowPath,
				instanceID: "test-instance",
				quiet:      true,
				skipPing:   tc.skipPing,
			}
			ctx := context.Background()
			result, err := plugin.evaluate(ctx, tc.request)
			if (err != nil) != tc.expectedError {
				t.Errorf("Expected error: %v, got: %v", tc.expectedError, err)
			}
			if result != tc.expectedResult {
				t.Errorf("Expected result: %v, got: %v", tc.expectedResult, result)
			}
		})
	}
}
