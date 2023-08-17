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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter/problemclient"
	"k8s.io/node-problem-detector/pkg/types"
	problemutil "k8s.io/node-problem-detector/pkg/util"

	v1 "k8s.io/api/core/v1"
	testclock "k8s.io/utils/clock/testing"
)

const heartbeatPeriod = 1 * time.Minute

func newTestManager() (*conditionManager, *problemclient.FakeProblemClient, *testclock.FakeClock) {
	fakeClient := problemclient.NewFakeProblemClient()
	fakeClock := testclock.NewFakeClock(time.Now())
	manager := NewConditionManager(fakeClient, fakeClock, heartbeatPeriod)
	return manager.(*conditionManager), fakeClient, fakeClock
}

func newTestCondition(condition string) types.Condition {
	return types.Condition{
		Type:       condition,
		Status:     types.True,
		Transition: time.Now(),
		Reason:     "TestReason",
		Message:    "test message",
	}
}

func TestNeedUpdates(t *testing.T) {
	m, _, _ := newTestManager()
	var c types.Condition
	for _, testCase := range []struct {
		name      string
		condition string
		update    bool
	}{
		{
			name:      "Init condition needs update",
			condition: "TestCondition",
			update:    true,
		},
		{
			name: "Same condition doesn't need update",
			// not set condition, the test will reuse the condition in last case.
			update: false,
		},
		{
			name:      "Same condition with different timestamp need update",
			condition: "TestCondition",
			update:    true,
		},
		{
			name:      "New condition needs update",
			condition: "TestConditionNew",
			update:    true,
		},
	} {
		tc := testCase
		t.Log(tc.name)
		if tc.condition != "" {
			// Guarantee that the time advances before creating a new condition.
			for now := time.Now(); now == time.Now(); {
			}
			c = newTestCondition(tc.condition)
		}
		m.UpdateCondition(c)
		assert.Equal(t, tc.update, m.needUpdates(), tc.name)
		assert.Equal(t, c, m.conditions[c.Type], tc.name)
	}
}

func TestGetConditions(t *testing.T) {
	m, _, _ := newTestManager()
	assert.Empty(t, m.GetConditions())
	testCondition1 := newTestCondition("TestCondition1")
	testCondition2 := newTestCondition("TestCondition2")
	m.UpdateCondition(testCondition1)
	m.UpdateCondition(testCondition2)
	assert.True(t, m.needUpdates())
	assert.Contains(t, m.GetConditions(), testCondition1)
	assert.Contains(t, m.GetConditions(), testCondition2)
}

func TestResync(t *testing.T) {
	m, fakeClient, fakeClock := newTestManager()
	condition := newTestCondition("TestCondition")
	m.conditions = map[string]types.Condition{condition.Type: condition}
	m.sync(context.Background())
	expected := []v1.NodeCondition{problemutil.ConvertToAPICondition(condition)}
	assert.Nil(t, fakeClient.AssertConditions(expected), "Condition should be updated via client")

	assert.False(t, m.needResync(), "Should not resync before resync period")
	fakeClock.Step(resyncPeriod)
	assert.False(t, m.needResync(), "Should not resync after resync period without resync needed")

	fakeClient.InjectError("SetConditions", fmt.Errorf("injected error"))
	m.sync(context.Background())

	assert.False(t, m.needResync(), "Should not resync before resync period")
	fakeClock.Step(resyncPeriod)
	assert.True(t, m.needResync(), "Should resync after resync period and resync is needed")
}

func TestHeartbeat(t *testing.T) {
	m, fakeClient, fakeClock := newTestManager()
	condition := newTestCondition("TestCondition")
	m.conditions = map[string]types.Condition{condition.Type: condition}
	m.sync(context.Background())
	expected := []v1.NodeCondition{problemutil.ConvertToAPICondition(condition)}
	assert.Nil(t, fakeClient.AssertConditions(expected), "Condition should be updated via client")

	assert.False(t, m.needHeartbeat(), "Should not heartbeat before heartbeat period")

	fakeClock.Step(heartbeatPeriod)
	assert.True(t, m.needHeartbeat(), "Should heartbeat after heartbeat period")
}
