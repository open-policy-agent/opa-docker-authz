package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
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
