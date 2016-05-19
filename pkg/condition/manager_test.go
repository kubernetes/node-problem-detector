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

package condition

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/node-problem-detector/pkg/problemclient"
	"k8s.io/node-problem-detector/pkg/types"
	problemutil "k8s.io/node-problem-detector/pkg/util"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util"
)

func newTestManager() (*conditionManager, *problemclient.FakeProblemClient, *util.FakeClock) {
	fakeClient := problemclient.NewFakeProblemClient()
	fakeClock := util.NewFakeClock(time.Now())
	manager := NewConditionManager(fakeClient, fakeClock)
	return manager.(*conditionManager), fakeClient, fakeClock
}

func newTestCondition() types.Condition {
	return types.Condition{
		Type:       "TestCondition",
		Status:     true,
		Transition: time.Now(),
		Reason:     "TestReason",
		Message:    "test message",
	}
}

func TestCheckUpdates(t *testing.T) {
	condition := newTestCondition()
	m, _, _ := newTestManager()
	m.UpdateCondition(condition)
	if !m.checkUpdates() {
		t.Error("expected checkUpdates to be true, got false")
	}
	if !reflect.DeepEqual(condition, m.conditions[condition.Type]) {
		t.Errorf("expected %+v, got %+v", condition, m.conditions[condition.Type])
	}
	if m.checkUpdates() {
		t.Error("expected checkUpdates to be false, got true")
	}
}

func TestSync(t *testing.T) {
	m, fakeClient, fakeClock := newTestManager()
	condition := newTestCondition()
	m.conditions = map[string]types.Condition{condition.Type: condition}
	m.sync()
	expected := []api.NodeCondition{problemutil.ConvertToAPICondition(condition)}
	err := fakeClient.AssertConditions(expected)
	if err != nil {
		t.Error(err)
	}
	if m.checkResync() {
		t.Error("expected checkResync to be false, got true")
	}
	fakeClock.Step(resyncPeriod)
	if !m.checkResync() {
		t.Error("expected checkResync to be true, got false")
	}
}
