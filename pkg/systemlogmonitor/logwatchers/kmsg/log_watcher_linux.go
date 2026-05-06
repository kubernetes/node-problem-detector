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
	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

const (
	// retryDelay is the time to wait before attempting to restart the kmsg parser.
	retryDelay = 5 * time.Second
)

type kernelLogWatcher struct {
	cfg       types.WatcherConfig
	startTime time.Time
	logCh     chan *logtypes.Log
	tomb      *tomb.Tomb

	kmsgParser kmsgparser.Parser
	// newParser creates a kmsgparser. Overridable in tests; defaults to kmsgparser.NewParser.
	newParser func() (kmsgparser.Parser, error)
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
		logCh:     make(chan *logtypes.Log, 100),
		newParser: kmsgparser.NewParser,
	}
}

var _ types.WatcherCreateFunc = NewKmsgWatcher

func (k *kernelLogWatcher) Watch() (<-chan *logtypes.Log, error) {
	if k.kmsgParser == nil {
		// nil-check to make mocking easier
		parser, err := k.newParser()
		if err != nil {
			return nil, fmt.Errorf("failed to create kmsg parser: %v", err)
		}
		k.kmsgParser = parser
	}

	go k.watchLoop()
	return k.logCh, nil
}

// Stop signals the watch loop to stop.
func (k *kernelLogWatcher) Stop() {
	k.tomb.Stop()
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
				klog.Error("Kmsg channel closed, attempting to restart kmsg parser")

				// Close the old parser
				if err := k.kmsgParser.Close(); err != nil {
					klog.Errorf("Failed to close kmsg parser: %v", err)
				}

				// Try to restart immediately. retryCreateParser() applies backoff only
				// after a failed NewParser() or SeekEnd() attempt.
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

			// Discard messages after now, lots of log files are not record year. 1 min cover log write latency
			if msg.Timestamp.After(time.Now().Add(time.Minute)) {
				klog.V(5).Infof("Throwing away msg %q after current time: %v < %v", msg.Message, msg.Timestamp, time.Now())
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
// It tries immediately first, then waits retryDelay between subsequent failures.
// On success, it seeks the new parser to the end of the kmsg ring buffer to
// avoid replaying messages that were already processed before the restart.
// Any messages written to kmsg between the old parser closing and the new
// parser being seeked are not delivered; this is preferable to replaying an
// entire ring buffer the watcher has already processed, especially when the
// restart was triggered by a kmsg flood.
// It returns the new message channel and true on success, or nil and false if stopping was signaled.
func (k *kernelLogWatcher) retryCreateParser() (<-chan kmsgparser.Message, bool) {
	for {
		parser, err := k.newParser()
		if err != nil {
			klog.Errorf("Failed to create new kmsg parser, retrying in %v: %v", retryDelay, err)
		} else if seekErr := parser.SeekEnd(); seekErr != nil {
			klog.Errorf("Failed to seek new kmsg parser to end, retrying in %v: %v", retryDelay, seekErr)
			if closeErr := parser.Close(); closeErr != nil {
				klog.Errorf("Failed to close kmsg parser after seek failure: %v", closeErr)
			}
		} else {
			k.kmsgParser = parser
			klog.Infof("Successfully restarted kmsg parser")
			return parser.Parse(), true
		}

		select {
		case <-k.tomb.Stopping():
			klog.Infof("Stop watching kernel log during restart attempt")
			return nil, false
		case <-time.After(retryDelay):
		}
	}
}
