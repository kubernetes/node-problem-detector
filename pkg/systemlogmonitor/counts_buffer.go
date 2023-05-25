/*
Copyright 2023 The Kubernetes Authors All rights reserved.

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

package systemlogmonitor

import (
	utilclock "code.cloudfoundry.org/clock"
	"fmt"
	"github.com/golang/glog"
	systemlogtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"time"
)

const (
	MaxCountRuleMatched = 128
	MaxTimeExpired      = time.Hour * 168
)

// CountRingBuffer is a struct to help store and calculate whether match the count and time period
// It is a ring buffer that when new log matched this rule , ring buffer add this timestamp in the end, and compare the time and count,
// decide to report error or not. The details in IsThresholdMatched() function.
type CountRingBuffer struct {
	data         []time.Time
	index        int
	length       int
	expirePeriod time.Duration
	clock        utilclock.Clock
}

func NewCountRingBuffer(thd int, ep time.Duration) *CountRingBuffer {
	return &CountRingBuffer{
		data:         make([]time.Time, thd),
		index:        0,
		length:       thd,
		expirePeriod: ep,
	}
}

// IsThresholdMatched compare the last one matched time and the top one matched time, if less expirePeriod set by user,
// report this error.
func (crb *CountRingBuffer) IsThresholdMatched() bool {
	tempIndex := crb.index
	crb.index = (crb.index + 1) % crb.length
    if crb.data[crb.index].IsZero() {
		crb.data[tempIndex] = time.Now()
		return false
	}
	crb.data[tempIndex] = time.Now()
	if crb.data[crb.index].Add(crb.expirePeriod).After(crb.data[tempIndex]) {
		return true
	}
	return false
}

// CountBuffer is used to record the count of pattern matched occur. If set CountThreshold and ExpirePeriod in config file.
// Every rule in config file will be auto created a CountRingBuffer this CountRingBuffer can decide whether report error considering
// the counts and period.
// One case is that, a user wants to set a rule but only 3 times in 10 minutes he consider it is a real error. Otherwise it is just a normal warning
// or can be auto-recovered. Now he can use this count buffer only if set CountThreshold = 3 and ExpirePeriod = 10m.
func NewCountBuffer(rules []systemlogtypes.Rule) map[string]*CountRingBuffer {
	if len(rules) == 0 {
		return nil
	}
	countBuffer := make(map[string]*CountRingBuffer)
	for _, rule := range rules {
		if rule.ExpirePeriod == "" {
			continue
		}
		expirePeriod, err := time.ParseDuration(rule.ExpirePeriod)
		if err != nil {
			glog.Errorf("%v", err)
			continue
		}
		if err := validateThreadAndTime(rule.CountThreshold, expirePeriod); err != nil {
			glog.Errorf("%v", err)
			continue
		}
		if rule.CountThreshold > 1 && expirePeriod > 0 {
			crb := NewCountRingBuffer(rule.CountThreshold, expirePeriod)
			countBuffer[rule.Reason] = crb
		}
	}
	return countBuffer
}

func validateThreadAndTime(countThreshold int, expirePeriod time.Duration) error {
	if countThreshold < 0 || countThreshold > MaxCountRuleMatched {
		return fmt.Errorf("Invalid Threshold in rule: %v", countThreshold)
	}
	if expirePeriod != 0 && expirePeriod > MaxTimeExpired {
		return fmt.Errorf("Invalid ExpirePeriod in rule: %v", expirePeriod)
	}
	return nil
}
