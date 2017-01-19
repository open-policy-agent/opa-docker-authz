// Copyright 2016 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"bytes"

	"github.com/docker/go-plugins-helpers/authorization"
	"github.com/fsnotify/fsnotify"
)

// DockerAuthZPlugin implements the authorization.Plugin interface.
// Every request received by the Docker daemon will be forwarded to the
// AuthZReq function. The AuthZReq function returns a response that indicates
// whether the request should be allowed or denied.
type DockerAuthZPlugin struct {
	opaURL string
}

// AuthZReq is called when the Docker daemon receives an API request.
// AuthZReq returns an authorization.Response that indicates whether the request should be
// allowed or denied.
func (p DockerAuthZPlugin) AuthZReq(r authorization.Request) authorization.Response {

	fmt.Println("Received request from Docker:", r)

	b, err := IsAllowed(p.opaURL, r)

	if b {
		return authorization.Response{Allow: true}
	} else if err != nil {
		return authorization.Response{Err: err.Error()}
	}

	return authorization.Response{Msg: "request rejected by administrative policy"}
}

// AuthZRes is called before the Docker daemon returns an API response. All responses
// are allowed.
func (p DockerAuthZPlugin) AuthZRes(r authorization.Request) authorization.Response {
	return authorization.Response{Allow: true}
}

// IsAllowed queries the policy that was loaded into OPA and returns (true, nil) if the
// request should be allowed. If the request is not allowed, b will be false and e will
// be set to indicate if an error occurred. This function "fails closed" meaning if an error
// occurs, the request will be rejected.
func IsAllowed(opaURL string, r authorization.Request) (b bool, e error) {

	// Query OPA to see if the request should be allowed.
	resp, err := QueryDataAPI(opaURL, "/opa/example/allow_request", r)
	if err != nil {
		return false, err
	}

	// If the request succeeded, the request should be allowed.
	if resp.StatusCode == 200 {
		return true, nil
	}

	// If the response is undefined, the request should be rejected.
	if IsUndefined(resp) {
		return false, nil
	}

	// Othewrise, an error occured so reject the request and include an error message.
	if resp.StatusCode == 404 {
		return false, fmt.Errorf("policy does not exist")
	}

	return false, fmt.Errorf("unexpected error: %v", resp)
}

// IsUndefined returns true if the http.Response resp from OPA indicates
// an undefined query result
func IsUndefined(resp *http.Response) bool {
	return resp.StatusCode == 404
}

// LoadPolicy reads the policy definition from the path f and upserts it into OPA.
func LoadPolicy(opaURL, f string) error {
	r, err := os.Open(f)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", opaURL+"/policies/example_policy", r)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		d := json.NewDecoder(resp.Body)
		var e map[string]interface{}
		if err := d.Decode(&e); err != nil {
			return err
		}
		return fmt.Errorf("policy PUT failed (code: %v): %v", e["Code"], e["Message"])
	}

	return nil
}

// WatchPolicy creates a filesystem watch on the path f and waits for changes. When the
// file changes, LoadPolicy is called with the path f.
func WatchPolicy(opaURL, f string) error {

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case evt := <-w.Events:
				if evt.Op&fsnotify.Write != 0 {
					if err := LoadPolicy(opaURL, f); err != nil {
						fmt.Println("Error reloading policy definition:", err)
					} else {
						fmt.Println("Reloaded policy definition.")
					}
				}
			}
		}
	}()

	if err := w.Add(f); err != nil {
		return err
	}

	return nil
}

// QueryDataAPI performs a POST against OPA's Data API. If successful, an
// http.Response is returned. The POST operation is a read-only operation that
// allows callers to supply an input document. OPA will include the input
// document when evaluating policies.
func QueryDataAPI(opaURL string, doc string, r authorization.Request) (*http.Response, error) {

	url := fmt.Sprintf("%s/data%s", opaURL, doc)
	body, err := encodeRequest(r)

	if err != nil {
		return nil, err
	}

	return http.Post(url, "application/json", body)
}

func encodeRequest(r authorization.Request) (io.Reader, error) {

	request := map[string]interface{}{
		"Headers":    r.RequestHeaders,
		"Path":       r.RequestURI,
		"Method":     r.RequestMethod,
		"Body":       r.RequestBody,
		"User":       r.User,
		"AuthMethod": r.UserAuthNMethod,
	}

	if r.RequestHeaders["Content-Type"] == "application/json" {
		var body interface{}
		if err := json.Unmarshal(r.RequestBody, &body); err != nil {
			return nil, err
		}
		request["Body"] = body
	}

	var body bytes.Buffer

	err := json.NewEncoder(&body).Encode(map[string]interface{}{
		"input": request,
	})

	if err != nil {
		return nil, err
	}

	return &body, nil
}

const (
	version = "0.1.4"
)

func main() {

	bindAddr := flag.String("bind-addr", ":8080", "sets the address the plugin will bind to")
	pluginName := flag.String("plugin-name", "opa-docker-authz", "sets the plugin name that will be registered with Docker")
	opaURL := flag.String("opa-url", "http://localhost:8181/v1", "sets the base URL of OPA's HTTP API")
	policyFile := flag.String("policy-file", "", "sets the path of the policy file to load")
	vers := flag.Bool("version", false, "print the version of the plugin")

	flag.Parse()

	if *vers {
		fmt.Println(version)
		os.Exit(0)
	}

	p := DockerAuthZPlugin{*opaURL}
	h := authorization.NewHandler(p)

	if *policyFile != "" {
		if err := LoadPolicy(*opaURL, *policyFile); err != nil {
			fmt.Println("Error while loading policy:", err)
			os.Exit(1)
		}

		if err := WatchPolicy(*opaURL, *policyFile); err != nil {
			fmt.Println("Error while starting watch:", err)
			os.Exit(1)
		}
	}

	fmt.Println("Starting server.")

	// No TLS configuration given for now.
	if err := h.ServeTCP(*pluginName, *bindAddr, nil); err != nil {
		fmt.Println("Error while serving HTTP:", err)
		os.Exit(1)
	}
}
