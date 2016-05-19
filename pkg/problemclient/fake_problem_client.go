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
	"sync"
	"time"

	"k8s.io/kubernetes/pkg/api"
)

// FakeProblemClient is a fake problem client for debug.
type FakeProblemClient struct {
	sync.Mutex
	conditions map[api.NodeConditionType]api.NodeCondition
	errors     map[string]error
}

// NewFakeProblemClient creates a new fake problem client.
func NewFakeProblemClient() *FakeProblemClient {
	return &FakeProblemClient{
		conditions: make(map[api.NodeConditionType]api.NodeCondition),
		errors:     make(map[string]error),
	}
}

// InjectError injects error to specific function.
func (f *FakeProblemClient) InjectError(fun string, err error) {
	f.Lock()
	defer f.Unlock()
	f.errors[fun] = err
}

// AssertConditions asserts that the internal conditions in fake problem client should match
// the expected conditions.
func (f *FakeProblemClient) AssertConditions(expected []api.NodeCondition) error {
	conditions := map[api.NodeConditionType]api.NodeCondition{}
	for _, condition := range expected {
		conditions[condition.Type] = condition
	}
	if !reflect.DeepEqual(conditions, f.conditions) {
		return fmt.Errorf("expected %+v, got %+v", conditions, f.conditions)
	}
	return nil
}

// SetConditions is a fake mimic of SetConditions, it only update the internal condition cache.
func (f *FakeProblemClient) SetConditions(conditions []api.NodeCondition, timeout time.Duration) error {
	f.Lock()
	defer f.Unlock()
	if err, ok := f.errors["SetConditions"]; ok {
		return err
	}
	for _, condition := range conditions {
		t := condition.Type
		if condition.Status == api.ConditionFalse {
			delete(f.conditions, t)
		} else {
			f.conditions[t] = condition
		}
	}
	return nil
}

// GetConditions is a fake mimic of GetConditions, it returns the conditions cached internally.
func (f *FakeProblemClient) GetConditions(types []api.NodeConditionType) ([]*api.NodeCondition, error) {
	f.Lock()
	defer f.Unlock()
	if err, ok := f.errors["GetConditions"]; ok {
		return nil, err
	}
	conditions := []*api.NodeCondition{}
	for _, t := range types {
		condition, ok := f.conditions[t]
		if ok {
			conditions = append(conditions, &condition)
		}
	}
	return conditions, nil
}

// Eventf does nothing now.
func (f *FakeProblemClient) Eventf(eventType string, source, reason, messageFmt string, args ...interface{}) {
}
