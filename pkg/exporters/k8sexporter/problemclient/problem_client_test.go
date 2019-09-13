/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package problemclient

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/record"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter/options"
)

const (
	testSource = "test"
	testNode   = "test-node"
)

func newFakeProblemClient() *nodeProblemClient {
	return &nodeProblemClient{
		nodeName: testNode,
		// There is no proper fake for *client.Client for now
		// TODO(random-liu): Add test for SetConditions when we have good fake for *client.Client
		clock:     &clock.FakeClock{},
		recorders: make(map[string]record.EventRecorder),
		nodeRef:   getNodeRef(testNode),
	}
}

func TestGeneratePatch(t *testing.T) {
	now := time.Now()
	update := []v1.NodeCondition{
		{
			Type:               "TestType1",
			Status:             v1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(now),
			Reason:             "TestReason1",
			Message:            "TestMessage1",
		},
		{
			Type:               "TestType2",
			Status:             v1.ConditionFalse,
			LastTransitionTime: metav1.NewTime(now),
			Reason:             "TestReason2",
			Message:            "TestMessage2",
		},
	}
	raw, err := json.Marshal(&update)
	assert.NoError(t, err)
	expectedPatch := []byte(fmt.Sprintf(`{"status":{"conditions":%s}}`, raw))

	patch, err := generatePatch(update)
	assert.NoError(t, err)
	if string(patch) != string(expectedPatch) {
		t.Errorf("expected patch %q, got %q", expectedPatch, patch)
	}
}

func TestEvent(t *testing.T) {
	fakeRecorder := record.NewFakeRecorder(1)
	client := newFakeProblemClient()
	client.recorders[testSource] = fakeRecorder
	client.Eventf(v1.EventTypeWarning, testSource, "test reason", "test message")
	expected := fmt.Sprintf("%s %s %s", v1.EventTypeWarning, "test reason", "test message")
	got := <-fakeRecorder.Events
	if expected != got {
		t.Errorf("expected event %q, got %q", expected, got)
	}
}

// TestSetNodeNameOrDie tests for permutations of nodename, hostname and hostnameoverride.
func TestSetNodeNameOrDie(t *testing.T) {
	hostName, err := os.Hostname()
	if err != nil {
		t.Errorf("Query hostname error: %v", err)
	}

	testCases := []struct {
		name             string
		wantedNodeName   string
		nodeNameEnv      string
		hostNameOverride string
	}{
		{
			name:             "Check hostname override only",
			wantedNodeName:   "hostname-override",
			nodeNameEnv:      "node-name-env",
			hostNameOverride: "hostname-override",
		},
		{
			name:             "Check hostname override and NODE_NAME env",
			wantedNodeName:   "node-name-env",
			nodeNameEnv:      "node-name-env",
			hostNameOverride: "",
		},
		{
			name:             "Check hostname override, NODE_NAME env and hostname",
			wantedNodeName:   hostName,
			nodeNameEnv:      "",
			hostNameOverride: "",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := os.Unsetenv("NODE_NAME")
			if err != nil {
				t.Errorf("Unset NODE_NAME env error: %v", err)
			}

			if test.nodeNameEnv != "" {
				err := os.Setenv("NODE_NAME", test.nodeNameEnv)
				if err != nil {
					t.Errorf("Set NODE_NAME env error: %v", err)
				}
			}

			k8sOptions := options.CommandLineOptions{HostnameOverride: test.hostNameOverride}

			client := nodeProblemClient{}
			client.setNodeNameOrDie(&k8sOptions)

			if client.nodeName != test.wantedNodeName {
				t.Errorf("Set node name error. Wanted: %v. Got: %v", test.wantedNodeName, client.nodeName)
			}
		})
	}
}
