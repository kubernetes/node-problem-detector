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
	"os"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	unversionedcore "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/unversioned"
	"k8s.io/kubernetes/pkg/client/record"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/types"
	"k8s.io/kubernetes/pkg/util"

	"github.com/golang/glog"
)

// Client is the interface of problem client
type Client interface {
	// GetConditions get all specifiec conditions of current node.
	GetConditions(conditionTypes []api.NodeConditionType) ([]*api.NodeCondition, error)
	// SetConditions set or update conditions of current node.
	// Notice that conditions with status api.ConditionFalse will be removed from the condition list, so that
	// we'll only have useful conditions in the condition list.
	SetConditions(conditions []api.NodeCondition, timeout time.Duration) error
	// Eventf reports the event.
	Eventf(eventType string, source, reason, messageFmt string, args ...interface{})
}

type nodeProblemClient struct {
	nodeName  string
	client    clientset.Interface
	clock     util.Clock
	recorders map[string]record.EventRecorder
	nodeRef   *api.ObjectReference
}

// NewClientOrDie creates a new problem client, panics if error occurs.
func NewClientOrDie() Client {
	c := &nodeProblemClient{clock: util.RealClock{}}
	cfg, err := restclient.InClusterConfig()
	if err != nil {
		panic(err)
	}
	// TODO(random-liu): Set QPS Limit
	c.client, err = clientset.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	// TODO(random-liu): Get node name from cloud provider
	c.nodeName, err = os.Hostname()
	if err != nil {
		panic(err)
	}
	c.nodeRef = getNodeRef(c.nodeName)
	c.recorders = make(map[string]record.EventRecorder)
	return c
}

func (c *nodeProblemClient) GetConditions(conditionTypes []api.NodeConditionType) ([]*api.NodeCondition, error) {
	node, err := c.client.Core().Nodes().Get(c.nodeName)
	if err != nil {
		return nil, err
	}
	conditions := []*api.NodeCondition{}
	for _, conditionType := range conditionTypes {
		for _, condition := range node.Status.Conditions {
			if condition.Type == conditionType {
				conditions = append(conditions, &condition)
			}
		}
	}
	return conditions, nil
}

func (c *nodeProblemClient) SetConditions(newConditions []api.NodeCondition, timeout time.Duration) error {
	for i := range newConditions {
		// Each time we update the conditions, we update the heart beat time
		newConditions[i].LastHeartbeatTime = unversioned.NewTime(c.clock.Now())
	}
	return c.updateNodeCondition(func(conditions []api.NodeCondition) []api.NodeCondition {
		for _, condition := range newConditions {
			if condition.Status == api.ConditionFalse {
				conditions = unsetCondition(condition.Type, conditions)
			} else {
				conditions = setCondition(condition, conditions)
			}
		}
		return conditions
	}, timeout)
}

func (c *nodeProblemClient) Eventf(eventType, source, reason, messageFmt string, args ...interface{}) {
	recorder, found := c.recorders[source]
	if !found {
		// TODO(random-liu): If needed use separate client and QPS limit for event.
		recorder = getEventRecorder(c.client, c.nodeName, source)
		c.recorders[source] = recorder
	}
	recorder.Eventf(c.nodeRef, eventType, reason, messageFmt, args...)
}

func unsetCondition(conditionType api.NodeConditionType, conditions []api.NodeCondition) []api.NodeCondition {
	result := []api.NodeCondition{}
	for _, condition := range conditions {
		if condition.Type != conditionType {
			result = append(result, condition)
		}
	}
	return result
}

func setCondition(condition api.NodeCondition, conditions []api.NodeCondition) []api.NodeCondition {
	found := false
	for i := range conditions {
		if conditions[i].Type == condition.Type {
			target := &conditions[i]
			*target = condition
			found = true
			break
		}
	}
	if !found {
		conditions = append(conditions, condition)
	}
	return conditions
}

func (c *nodeProblemClient) updateNodeCondition(updateFunc func([]api.NodeCondition) []api.NodeCondition, timeout time.Duration) error {
	updateTime := c.clock.Now()
	for {
		node, err := c.client.Core().Nodes().Get(c.nodeName)
		if err != nil {
			return err
		}
		node.Status.Conditions = updateFunc(node.Status.Conditions)
		_, err = c.client.Core().Nodes().UpdateStatus(node)
		if err != nil {
			if errors.IsConflict(err) {
				glog.Warningf("Conflicting update node status for node %q, will retry soon: %v", c.nodeName, err)
				if c.clock.Now().Sub(updateTime) >= timeout {
					return timeoutError{node: c.nodeName, timeout: timeout}
				}
				continue
			}
			return err
		}
		return nil
	}
}

// getEventRecorder generates a recorder for specific node name and source.
func getEventRecorder(c clientset.Interface, nodeName, source string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(api.EventSource{Component: source, Host: nodeName})
	eventBroadcaster.StartRecordingToSink(&unversionedcore.EventSinkImpl{Interface: c.Core().Events("")})
	return recorder
}

func getNodeRef(nodeName string) *api.ObjectReference {
	return &api.ObjectReference{
		Kind:      "Node",
		Name:      nodeName,
		UID:       types.UID(nodeName),
		Namespace: "",
	}
}

// timeoutError is the error returned by problem client when condition update timeout.
type timeoutError struct {
	node    string
	timeout time.Duration
}

func (e timeoutError) Error() string {
	return fmt.Sprintf("update condition for node %q timeout %s", e.node, e.timeout)
}

// IsErrTimeout checks whether a given error is timeout error.
func IsErrTimeout(err error) bool {
	_, ok := err.(timeoutError)
	return ok
}
