/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package options

import (
	"os"
	"testing"
)

type options struct {
	Nodename         string
	HostnameOverride string
}

// TestSetNodeNameOrDie tests for permutations of nodename, hostname and hostnameoverride.
func TestSetNodeNameOrDie(t *testing.T) {
	option := map[string]struct {
		Expected         options
		ObtainedNodeName string
	}{
		"Check Node and HostnameOverride only": {
			Expected: options{
				Nodename:         "my-node-name",
				HostnameOverride: "override",
			},
		},
		"Check Nodename only": {
			Expected: options{
				Nodename:         "my-node-name",
				HostnameOverride: "",
			},
		},

		"Check HostnameOverride only": {
			Expected: options{
				Nodename:         "",
				HostnameOverride: "override",
			},
		},
		"Check empty": {
			Expected: options{
				Nodename:         "",
				HostnameOverride: "",
			},
		},
	}

	orig_node_name := os.Getenv("NODE_NAME")
	orig_host_name, err := os.Hostname()
	if err != nil {
		t.Errorf("Unable to get hostname")
	}

	for str, opt := range option {

		// Setting with expected(desired) NODE_NAME env.
		err = os.Setenv("NODE_NAME", opt.Expected.Nodename)
		if err != nil {
			t.Errorf("Unable to set env NODE_NAME")
		}

		npdObj := NewNodeProblemDetectorOptions()

		// Setting with expected(desired) HostnameOverride.
		npdObj.HostnameOverride = opt.Expected.HostnameOverride

		npdObj.SetNodeNameOrDie()
		opt.ObtainedNodeName = npdObj.NodeName

		// Setting back the original node name.
		err = os.Setenv("NODE_NAME", orig_node_name)
		if err != nil {
			t.Errorf("Unable to set original : env NODE_NAME")
		}

		// Checking for obtained node name.
		if opt.ObtainedNodeName != opt.Expected.HostnameOverride &&
			opt.ObtainedNodeName != opt.Expected.Nodename &&
			opt.ObtainedNodeName != orig_host_name {
			t.Errorf("Error at : %+v", str)
			t.Errorf("Wanted: %+v. \nGot: %+v", opt.Expected.Nodename, opt.ObtainedNodeName)
		}

	}

}
