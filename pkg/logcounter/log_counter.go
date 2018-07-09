/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package logcounter

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/clock"

	"k8s.io/node-problem-detector/cmd/logcounter/options"
	"k8s.io/node-problem-detector/pkg/logcounter/types"
	"k8s.io/node-problem-detector/pkg/systemlogmonitor"
	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/kmsg"
	watchertypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	systemtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
)

const (
	bufferSize = 1000
	timeout    = 1 * time.Second
)

type logCounter struct {
	logCh   <-chan *systemtypes.Log
	buffer  systemlogmonitor.LogBuffer
	pattern string
	clock   clock.Clock
}

func NewKmsgLogCounter(options *options.LogCounterOptions) (types.LogCounter, error) {
	watcher := kmsg.NewKmsgWatcher(watchertypes.WatcherConfig{Lookback: options.Lookback})
	logCh, err := watcher.Watch()
	if err != nil {
		return nil, fmt.Errorf("error watching kmsg: %v", err)
	}
	return &logCounter{
		logCh:   logCh,
		buffer:  systemlogmonitor.NewLogBuffer(bufferSize),
		pattern: options.Pattern,
		clock:   clock.RealClock{},
	}, nil
}

func (e *logCounter) Count() (count int) {
	start := e.clock.Now()
	for {
		select {
		case log := <-e.logCh:
			// We only want to count events up until the time at which we started.
			// Otherwise we would run forever
			if start.Before(log.Timestamp) {
				return
			}
			e.buffer.Push(log)
			if len(e.buffer.Match(e.pattern)) != 0 {
				count++
			}
		case <-e.clock.After(timeout):
			// Don't block forever if we do not get any new messages
			return
		}
	}
}
