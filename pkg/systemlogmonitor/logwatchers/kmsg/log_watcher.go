/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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

package kmsg

import (
	"fmt"
	"strings"
	"time"

	utilclock "code.cloudfoundry.org/clock"
	"github.com/euank/go-kmsg-parser/kmsgparser"
	"github.com/golang/glog"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

type kernelLogWatcher struct {
	cfg   types.WatcherConfig
	logCh chan *logtypes.Log
	tomb  *tomb.Tomb

	kmsgParser kmsgparser.Parser
	clock      utilclock.Clock
}

// NewKmsgWatcher creates a watcher which will read messages from /dev/kmsg
func NewKmsgWatcher(cfg types.WatcherConfig) types.LogWatcher {
	return &kernelLogWatcher{
		cfg:  cfg,
		tomb: tomb.NewTomb(),
		// Arbitrary capacity
		logCh: make(chan *logtypes.Log, 100),
		clock: utilclock.NewClock(),
	}
}

var _ types.WatcherCreateFunc = NewKmsgWatcher

func (k *kernelLogWatcher) Watch() (<-chan *logtypes.Log, error) {
	if k.kmsgParser == nil {
		// nil-check to make mocking easier
		parser, err := kmsgparser.NewParser()
		if err != nil {
			return nil, fmt.Errorf("failed to create kmsg parser: %v", err)
		}
		k.kmsgParser = parser
	}

	lookback, err := time.ParseDuration(k.cfg.Lookback)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lookback duration %q: %v", k.cfg.Lookback, err)
	}

	go k.watchLoop(lookback)
	return k.logCh, nil
}

// Stop closes the kmsgparser
func (k *kernelLogWatcher) Stop() {
	k.kmsgParser.Close()
	k.tomb.Stop()
}

// watchLoop is the main watch loop of kernel log watcher.
func (k *kernelLogWatcher) watchLoop(lookback time.Duration) {
	defer func() {
		close(k.logCh)
		k.tomb.Done()
	}()
	kmsgs := k.kmsgParser.Parse()

	for {
		select {
		case <-k.tomb.Stopping():
			glog.Infof("Stop watching kernel log")
			if err := k.kmsgParser.Close(); err != nil {
				glog.Errorf("Failed to close kmsg parser: %v", err)
			}
			return
		case msg := <-kmsgs:
			glog.V(5).Infof("got kernel message: %+v", msg)
			if msg.Message == "" {
				continue
			}

			// Discard too old messages
			if k.clock.Since(msg.Timestamp) > lookback {
				glog.V(5).Infof("Throwing away msg %v for being too old: %v > %v", msg.Message, msg.Timestamp.String(), lookback.String())
				continue
			}

			k.logCh <- &logtypes.Log{
				Message:   strings.TrimSpace(msg.Message),
				Timestamp: msg.Timestamp,
			}
		}
	}
}
