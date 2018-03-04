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
	"fmt"
	"os"
	"os/exec"
	"testing"
)

type Options struct {
	Nodename         string
	Hostname         string
	HostnameOverride string
}

// TestSetNodeNameOrDie tests for permutations of nodename, hostname and hostnameoverride
func TestSetNodeNameOrDie(t *testing.T) {
	options := map[string]struct {
		Expected         Options
		ObtainedNodeName string
	}{
		"Check Node and HostnameOverride only": {
			Expected: Options{
				Nodename:         "my-node-name",
				Hostname:         "",
				HostnameOverride: "override",
			},
		},
		"Check Nodename only": {
			Expected: Options{
				Nodename:         "my-node-name",
				Hostname:         "",
				HostnameOverride: "",
			},
		},

		"Check HostnameOverride only": {
			Expected: Options{
				Nodename:         "",
				Hostname:         "",
				HostnameOverride: "override",
			},
		},
		"Check Hostname only": {
			Expected: Options{
				Nodename:         "",
				Hostname:         "my-host-name",
				HostnameOverride: "",
			},
		},
		"Check Node, host and HostnameOverride only": {
			Expected: Options{
				Nodename:         "my-node-name",
				Hostname:         "my-host-name",
				HostnameOverride: "override",
			},
		},

		"Check Host and HostnameOverride only": {
			Expected: Options{
				Nodename:         "",
				Hostname:         "my-host-name",
				HostnameOverride: "override",
			},
		},

		"Check Node and hostname": {
			Expected: Options{
				Nodename:         "my-node-name",
				Hostname:         "my-host-name",
				HostnameOverride: "",
			},
		},
	}

	orig_node_name := os.Getenv("NODE_NAME")
	orig_host_name, err := os.Hostname()
	if err != nil {
		fmt.Println("Unable to get hostname")
	}

	for str, opt := range options {
		if opt.Expected.Nodename != "" {
			err = os.Setenv("NODE_NAME", opt.Expected.Nodename)
			if err != nil {
				t.Errorf("Unable to set env NODE_NAME")
			}
		}

		if opt.Expected.Hostname != "" {
			// Changing hostname
			cmd := exec.Command("hostname", opt.Expected.Hostname)
			_, err := cmd.CombinedOutput()
			if err != nil {
				// If changing hostname requires admin privilege
				cmd = exec.Command("sudo", "hostname", opt.Expected.Hostname)
				_, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Println("Unable to change hostname")
					return
				}
			}
		}

		npdObj := NewNodeProblemDetectorOptions()
		npdObj.HostnameOverride = opt.Expected.HostnameOverride
		npdObj.SetNodeNameOrDie()
		opt.ObtainedNodeName = npdObj.NodeName

		if opt.ObtainedNodeName != opt.Expected.HostnameOverride &&
			opt.ObtainedNodeName != opt.Expected.Nodename &&
			opt.ObtainedNodeName != opt.Expected.Hostname {
			t.Errorf("Error at : %+v", str)
			t.Errorf("Wanted: %+v. \nGot: %+v", opt.Expected.Nodename, opt.ObtainedNodeName)
		}

		err = os.Setenv("NODE_NAME", "")
		if err != nil {
			t.Errorf("Unable to set env NODE_NAME empty")
		}

	}
	// Setting back the original attributes
	err = os.Setenv("NODE_NAME", orig_node_name)
	if err != nil {
		fmt.Println("Unable to set original : env NODE_NAME")
	}
	cmd := exec.Command("sudo", "hostname", orig_host_name)
	_, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Unable to set hostname")
	}

}
