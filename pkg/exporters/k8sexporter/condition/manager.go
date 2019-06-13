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
	"sync"
	"time"

	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter/problemclient"
	"k8s.io/node-problem-detector/pkg/types"
	problemutil "k8s.io/node-problem-detector/pkg/util"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/clock"

	"github.com/golang/glog"
)

const (
	// updatePeriod is the period at which condition manager checks update.
	updatePeriod = 1 * time.Second
	// resyncPeriod is the period at which condition manager does resync, only updates when needed.
	resyncPeriod = 10 * time.Second
	// heartbeatPeriod is the period at which condition manager does forcibly sync with apiserver.
	heartbeatPeriod = 1 * time.Minute
)

// ConditionManager synchronizes node conditions with the apiserver with problem client.
// It makes sure that:
// 1) Node conditions are updated to apiserver as soon as possible.
// 2) Node problem detector won't flood apiserver.
// 3) No one else could change the node conditions maintained by node problem detector.
// ConditionManager checks every updatePeriod to see whether there is node condition update. If there are any,
// it will synchronize with the apiserver. This addresses 1) and 2).
// ConditionManager synchronizes with apiserver every resyncPeriod no matter there is node condition update or
// not. This addresses 3).
type ConditionManager interface {
	// Start starts the condition manager.
	Start()
	// UpdateCondition updates a specific condition.
	UpdateCondition(types.Condition)
	// GetConditions returns all current conditions.
	GetConditions() []types.Condition
}

type conditionManager struct {
	// Only 2 fields will be accessed by more than one goroutines at the same time:
	// * `updates`: updates will be written by random caller and the sync routine,
	// so it needs to be protected by write lock in both `UpdateCondition` and
	// `needUpdates`.
	// * `conditions`: conditions will only be written in the sync routine, but
	// it will be read by random caller and the sync routine. So it needs to be
	// protected by write lock in `needUpdates` and read lock in `GetConditions`.
	// No lock is needed in `sync`, because it is in the same goroutine with the
	// write operation.
	sync.RWMutex
	clock        clock.Clock
	latestTry    time.Time
	resyncNeeded bool
	client       problemclient.Client
	updates      map[string]types.Condition
	conditions   map[string]types.Condition
}

// NewConditionManager creates a condition manager.
func NewConditionManager(client problemclient.Client, clock clock.Clock) ConditionManager {
	return &conditionManager{
		client:     client,
		clock:      clock,
		updates:    make(map[string]types.Condition),
		conditions: make(map[string]types.Condition),
	}
}

func (c *conditionManager) Start() {
	go c.syncLoop()
}

func (c *conditionManager) UpdateCondition(condition types.Condition) {
	c.Lock()
	defer c.Unlock()
	// New node condition will override the old condition, because we only need the newest
	// condition for each condition type.
	c.updates[condition.Type] = condition
}

func (c *conditionManager) GetConditions() []types.Condition {
	c.RLock()
	defer c.RUnlock()
	var conditions []types.Condition
	for _, condition := range c.conditions {
		conditions = append(conditions, condition)
	}
	return conditions
}

func (c *conditionManager) syncLoop() {
	updateCh := c.clock.Tick(updatePeriod)
	for {
		select {
		case <-updateCh:
			if c.needUpdates() || c.needResync() || c.needHeartbeat() {
				c.sync()
			}
		}
	}
}

// needUpdates checks whether there are recent updates.
func (c *conditionManager) needUpdates() bool {
	c.Lock()
	defer c.Unlock()
	needUpdate := false
	for t, update := range c.updates {
		if !reflect.DeepEqual(c.conditions[t], update) {
			needUpdate = true
			c.conditions[t] = update
		}
		delete(c.updates, t)
	}
	return needUpdate
}

// needResync checks whether a resync is needed.
func (c *conditionManager) needResync() bool {
	// Only update when resync is needed.
	return c.clock.Now().Sub(c.latestTry) >= resyncPeriod && c.resyncNeeded
}

// needHeartbeat checks whether a forcible heartbeat is needed.
func (c *conditionManager) needHeartbeat() bool {
	return c.clock.Now().Sub(c.latestTry) >= heartbeatPeriod
}

// sync synchronizes node conditions with the apiserver.
func (c *conditionManager) sync() {
	c.latestTry = c.clock.Now()
	c.resyncNeeded = false
	conditions := []v1.NodeCondition{}
	for i := range c.conditions {
		conditions = append(conditions, problemutil.ConvertToAPICondition(c.conditions[i]))
	}
	if err := c.client.SetConditions(conditions); err != nil {
		// The conditions will be updated again in future sync
		glog.Errorf("failed to update node conditions: %v", err)
		c.resyncNeeded = true
		return
	}
}
