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
	hostName, err := os.Hostname()
	if err != nil {
		t.Errorf("Query hostname error: %v", err)
	}

	uts := map[string]struct {
		WantedNodeName string
		Meta           options
	}{
		"Check hostname override only": {
			WantedNodeName: "hostname-override",
			Meta: options{
				Nodename:         "node-name-env",
				HostnameOverride: "hostname-override",
			},
		},
		"Check hostname override and NDDE_NAME env": {
			WantedNodeName: "node-name-env",
			Meta: options{
				Nodename:         "node-name-env",
				HostnameOverride: "",
			},
		},
		"Check hostname override, NODE_NAME env and hostname": {
			WantedNodeName: hostName,
			Meta: options{
				Nodename:         "",
				HostnameOverride: "",
			},
		},
	}

	for desc, ut := range uts {
		err := os.Unsetenv("NODE_NAME")
		if err != nil {
			t.Errorf("Desc: %v. Unset NODE_NAME env error: %v", desc, err)
		}

		if len(ut.Meta.Nodename) != 0 {
			err := os.Setenv("NODE_NAME", ut.Meta.Nodename)
			if err != nil {
				t.Errorf("Desc: %v. Set NODE_NAME env error: %v", desc, err)
			}
		}

		npdOpts := NewNodeProblemDetectorOptions()
		npdOpts.HostnameOverride = ut.Meta.HostnameOverride
		npdOpts.SetNodeNameOrDie()

		if npdOpts.NodeName != ut.WantedNodeName {
			t.Errorf("Desc: %v. Set node name error. Wanted: %v. Got: %v", desc, ut.WantedNodeName, npdOpts.NodeName)
		}
	}
}
