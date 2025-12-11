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

	"github.com/euank/go-kmsg-parser/kmsgparser"
	klog "k8s.io/klog/v2"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

const (
	// retryDelay is the time to wait before attempting to restart the kmsg parser.
	retryDelay = 5 * time.Second

	// RestartOnErrorKey is the configuration key to enable restarting
	// the kmsg parser when the channel closes due to an error.
	RestartOnErrorKey = "restartOnError"
)

type kernelLogWatcher struct {
	cfg       types.WatcherConfig
	startTime time.Time
	logCh     chan *logtypes.Log
	tomb      *tomb.Tomb

	kmsgParser kmsgparser.Parser
}

// NewKmsgWatcher creates a watcher which will read messages from /dev/kmsg
func NewKmsgWatcher(cfg types.WatcherConfig) types.LogWatcher {
	uptime, err := util.GetUptimeDuration()
	if err != nil {
		klog.Fatalf("failed to get uptime: %v", err)
	}
	startTime, err := util.GetStartTime(time.Now(), uptime, cfg.Lookback, cfg.Delay)
	if err != nil {
		klog.Fatalf("failed to get start time: %v", err)
	}

	return &kernelLogWatcher{
		cfg:       cfg,
		startTime: startTime,
		tomb:      tomb.NewTomb(),
		// Arbitrary capacity
		logCh: make(chan *logtypes.Log, 100),
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

	go k.watchLoop()
	return k.logCh, nil
}

// Stop closes the kmsgparser
func (k *kernelLogWatcher) Stop() {
	if err := k.kmsgParser.Close(); err != nil {
		klog.Errorf("Failed to close kmsg parser: %v", err)
	}
	k.tomb.Stop()
}

// restartOnError checks if the restart on error configuration is enabled.
func (k *kernelLogWatcher) restartOnError() bool {
	value, exists := k.cfg.PluginConfig[RestartOnErrorKey]
	return exists && value == "true"
}

// watchLoop is the main watch loop of kernel log watcher.
func (k *kernelLogWatcher) watchLoop() {
	kmsgs := k.kmsgParser.Parse()
	defer func() {
		if err := k.kmsgParser.Close(); err != nil {
			klog.Errorf("Failed to close kmsg parser: %v", err)
		}
		close(k.logCh)
		k.tomb.Done()
	}()

	for {
		select {
		case <-k.tomb.Stopping():
			klog.Infof("Stop watching kernel log")
			return
		case msg, ok := <-kmsgs:
			if !ok {
				klog.Error("Kmsg channel closed")

				// Only attempt to restart if configured to do so
				if !k.restartOnError() {
					klog.Infof("Restart on error not enabled, stopping watcher")
					return
				}

				klog.Infof("Attempting to restart kmsg parser")

				// Close the old parser
				if err := k.kmsgParser.Close(); err != nil {
					klog.Errorf("Failed to close kmsg parser: %v", err)
				}

				// Try to restart with backoff
				var restarted bool
				kmsgs, restarted = k.retryCreateParser()
				if !restarted {
					// Stopping was signaled
					return
				}
				continue
			}
			klog.V(5).Infof("got kernel message: %+v", msg)
			if msg.Message == "" {
				continue
			}

			// Discard messages before start time.
			if msg.Timestamp.Before(k.startTime) {
				klog.V(5).Infof("Throwing away msg %q before start time: %v < %v", msg.Message, msg.Timestamp, k.startTime)
				continue
			}

			k.logCh <- &logtypes.Log{
				Message:   strings.TrimSpace(msg.Message),
				Timestamp: msg.Timestamp,
			}
		}
	}
}

// retryCreateParser attempts to create a new kmsg parser.
// It returns the new message channel and true on success, or nil and false if stopping was signaled.
func (k *kernelLogWatcher) retryCreateParser() (<-chan kmsgparser.Message, bool) {
	for {
		select {
		case <-k.tomb.Stopping():
			klog.Infof("Stop watching kernel log during restart attempt")
			return nil, false
		case <-time.After(retryDelay):
		}

		parser, err := kmsgparser.NewParser()
		if err != nil {
			klog.Errorf("Failed to create new kmsg parser, retrying in %v: %v", retryDelay, err)
			continue
		}

		k.kmsgParser = parser
		klog.Infof("Successfully restarted kmsg parser")
		return parser.Parse(), true
	}
}
