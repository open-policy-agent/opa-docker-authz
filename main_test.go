package main

import (
	"encoding/json"
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
	tests := []struct {
		statement string
		input     string
		expected  []BindMount
	}{
		{
			statement: "parse a simple bind list",
			input:     `{ "HostConfig": { "Binds" : [ "/var:/home", "volume:/var/lib/app:ro" ] } }`,
			expected:  []BindMount{{"/var", false, ""}},
		},
		{
			statement: "parse the readonly attribute",
			input:     `{ "HostConfig": { "Binds" : [ "/var:/home:ro", "/home/user:/mnt:rw" ] } }`,
			expected:  []BindMount{{"/var", true, ""}, {"/home/user", false, ""}},
		},
		{
			statement: "handle when neither bind nor mounts provided",
			input:     `{ "HostConfig": {{} }`,
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
			expected: []BindMount{{"/var", false, ""}},
		},
		{
			statement: "parse a readonly mount list",
			input: `{ "HostConfig": { "Mounts" : [ 
				{ "Source": "/var", "Target": "/mnt", "Type": "bind", "ReadOnly": true },
				{ "Source": "/home", "Target": "/home", "Type": "bind" }
				] } }`,
			expected: []BindMount{{"/var", true, ""}, {"/home", false, ""}},
		},
		{
			statement: "ignore an invalid mount list",
			input: `{ "HostConfig": { "Mounts" : [ 
				{ "Source": "/var", "Target": "/mnt", "Type": "bind", "ReadOnly": true },
				{ "Source1": "/home", "Target": "/home", "Type": "bind" }
				] } }`,
			expected: []BindMount{{"/var", true, ""}},
		},
		{
			statement: "ignore a mount list of the wrong type, whlile reading binds",
			input: `{ "HostConfig": { "Binds": ["/var:/mnt/var:ro","/home:/home"],
				"Mounts" : null } }`,
			expected: []BindMount{{"/var", true, ""}, {"/home", false, ""}},
		},
	}

	for _, tc := range tests {
		t.Run("listBindMounts should "+tc.statement, func(t *testing.T) {
			var body map[string]interface{}
			json.Unmarshal([]byte(tc.input), &body)

			result := listBindMounts(body)
			if len(result) > 0 && len(tc.expected) > 0 && !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}
