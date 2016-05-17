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
	"fmt"
	"reflect"
	"testing"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	"k8s.io/kubernetes/pkg/client/record"
	"k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util"
)

const (
	testSource = "test"
	testNode   = "test-node"
)

func newFakeProblemClient(fakeClient *fake.Clientset) *nodeProblemClient {
	return &nodeProblemClient{
		nodeName:  testNode,
		client:    fakeClient,
		clock:     &util.FakeClock{},
		recorders: make(map[string]record.EventRecorder),
		nodeRef:   getNodeRef(testNode),
	}
}

func newFakeNode(conditions []api.NodeCondition) *api.Node {
	node := &api.Node{}
	node.Name = testNode
	node.Status = api.NodeStatus{Conditions: conditions}
	return node
}

type action struct {
	verb        string
	resource    string
	subresource string
}

func TestSetConditions(t *testing.T) {
	now := time.Now()
	expectedActions := []action{
		{
			verb:     "get",
			resource: "nodes",
		},
		{
			verb:        "update",
			resource:    "nodes",
			subresource: "status",
		},
	}
	for _, test := range []struct {
		init     []api.NodeCondition
		update   []api.NodeCondition
		expected []api.NodeCondition
	}{
		// Init condition with the same type should be override
		{
			init: []api.NodeCondition{
				{
					Type:   "TestType",
					Status: api.ConditionTrue,
				},
			},
			update: []api.NodeCondition{
				{
					Type:               "TestType",
					Status:             api.ConditionTrue,
					LastTransitionTime: unversioned.NewTime(now),
					Reason:             "TestReason",
					Message:            "TestMessage",
				},
			},
			expected: []api.NodeCondition{
				{
					// LastHeartbeatTime should be updated in SetConditions
					Type:               "TestType",
					Status:             api.ConditionTrue,
					LastHeartbeatTime:  unversioned.NewTime(now),
					LastTransitionTime: unversioned.NewTime(now),
					Reason:             "TestReason",
					Message:            "TestMessage",
				},
			},
		},
		// Init condition with different type should be kept
		{
			init: []api.NodeCondition{
				{
					Type:               "InitType",
					Status:             api.ConditionTrue,
					LastTransitionTime: unversioned.NewTime(now),
					Reason:             "InitReason",
					Message:            "InitMessage",
				},
			},
			update: []api.NodeCondition{
				{
					Type:               "TestType",
					Status:             api.ConditionTrue,
					LastTransitionTime: unversioned.NewTime(now),
					Reason:             "TestReason",
					Message:            "TestMessage",
				},
			},
			expected: []api.NodeCondition{
				{
					Type:               "InitType",
					Status:             api.ConditionTrue,
					LastTransitionTime: unversioned.NewTime(now),
					Reason:             "InitReason",
					Message:            "InitMessage",
				},
				{
					// LastHeartbeatTime should be updated in SetConditions
					Type:               "TestType",
					Status:             api.ConditionTrue,
					LastHeartbeatTime:  unversioned.NewTime(now),
					LastTransitionTime: unversioned.NewTime(now),
					Reason:             "TestReason",
					Message:            "TestMessage",
				},
			},
		},
		// Condition with false status should be removed
		{
			init: []api.NodeCondition{
				{
					Type:               "TestType",
					Status:             api.ConditionTrue,
					LastHeartbeatTime:  unversioned.NewTime(now),
					LastTransitionTime: unversioned.NewTime(now),
					Reason:             "TestReason",
					Message:            "TestMessage",
				},
			},
			update: []api.NodeCondition{
				{
					Type:   "TestType",
					Status: api.ConditionFalse,
				},
			},
			expected: []api.NodeCondition{},
		},
	} {
		fakeClient := fake.NewSimpleClientset(newFakeNode(test.init))
		client := newFakeProblemClient(fakeClient)
		clock := client.clock.(*util.FakeClock)
		clock.SetTime(now)

		client.SetConditions(test.update, 10*time.Second)

		// The actions should match the expected actions
		actions := fakeClient.Actions()
		if len(expectedActions) != len(actions) {
			t.Errorf("expected actions %+v, got %+v", expectedActions, fakeClient.Actions())
			continue
		}
		for i, a := range actions {
			if !a.Matches(expectedActions[i].verb, expectedActions[i].resource) || a.GetSubresource() != expectedActions[i].subresource {
				t.Errorf("expected action %+v, got %+v", expectedActions[i], a)
			}
		}
		// The last action should be an update
		a, ok := actions[len(actions)-1].(core.UpdateAction)
		if !ok {
			t.Errorf("expected the last action to be update, got %+v", actions[len(actions)-1])
		}
		// The updated node conditions should match the expected conditions
		node, ok := a.GetObject().(*api.Node)
		if !ok {
			t.Errorf("expected the update object to be node, got %+v", a.GetObject())
		}
		if !api.Semantic.DeepEqual(test.expected, node.Status.Conditions) {
			t.Errorf("expected conditions %+v, got %+v", test.expected, node.Status.Conditions)
		}
	}
}

func TestSetConditionsError(t *testing.T) {
	timeout := time.Duration(0)
	node := newFakeNode([]api.NodeCondition{})
	for c, test := range []struct {
		errMap      map[string]error
		expectedErr error
	}{
		{
			// Get error
			errMap:      map[string]error{"get": fmt.Errorf("get error")},
			expectedErr: fmt.Errorf("get error"),
		},
		{
			// Update error
			errMap:      map[string]error{"update": fmt.Errorf("update error")},
			expectedErr: fmt.Errorf("update error"),
		},
		{
			// Timeout error
			errMap: map[string]error{
				"update": &errors.StatusError{ErrStatus: unversioned.Status{Reason: unversioned.StatusReasonConflict}},
			},
			expectedErr: timeoutError{node: testNode, timeout: timeout},
		},
		{
			// No error
			errMap:      map[string]error{},
			expectedErr: nil,
		},
	} {
		fakeClient := &fake.Clientset{}
		client := newFakeProblemClient(fakeClient)
		fakeClient.AddReactor("get", "nodes", func(action core.Action) (bool, runtime.Object, error) {
			return true, node, test.errMap["get"]
		})
		fakeClient.AddReactor("update", "nodes", func(action core.Action) (bool, runtime.Object, error) {
			return true, node, test.errMap["update"]
		})
		err := client.SetConditions([]api.NodeCondition{}, timeout)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("case %d: expected error %v, got %v", c+1, test.expectedErr, err)
		}
	}
}

func TestEvent(t *testing.T) {
	fakeRecorder := record.NewFakeRecorder(1)
	client := newFakeProblemClient(&fake.Clientset{})
	client.recorders[testSource] = fakeRecorder
	client.Eventf(api.EventTypeWarning, testSource, "test reason", "test message")
	expected := fmt.Sprintf("%s %s %s", api.EventTypeWarning, "test reason", "test message")
	got := <-fakeRecorder.Events
	if expected != got {
		t.Errorf("expected event %q, got %q", expected, got)
	}
}
