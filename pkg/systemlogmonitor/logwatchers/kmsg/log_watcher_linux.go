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
	reviveRetries  = 10
	reviveDuration = 5 * time.Second
)

type kernelLogWatcher struct {
	cfg       types.WatcherConfig
	startTime time.Time
	logCh     chan *logtypes.Log
	tomb      *tomb.Tomb

	kmsgParser  kmsgparser.Parser
	reviveCount int
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
		logCh:       make(chan *logtypes.Log, 100),
		reviveCount: 0,
	}
}

var _ types.WatcherCreateFunc = NewKmsgWatcher

func (k *kernelLogWatcher) Watch() (<-chan *logtypes.Log, error) {
	if err := k.SetKmsgParser(); err != nil {
		return nil, err
	}
	go k.watchLoop()
	return k.logCh, nil
}

// Stop closes the kmsgparser
func (k *kernelLogWatcher) Stop() {
	k.kmsgParser.Close()
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
				if val, ok := k.cfg.PluginConfig["revive"]; ok && val == "true" {
					k.reviveMyself()
				}
				klog.Error("Kmsg channel closed")
				return
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

// create a new kmsg parser and sets it to the watcher.
func (k *kernelLogWatcher) SetKmsgParser() error {
	parser, err := kmsgparser.NewParser()
	if err != nil {
		return fmt.Errorf("failed to create kmsg parser: %v", err)
	}
	k.kmsgParser = parser
	return nil
}

// revive ourselves if the kmsg channel is closed
// close the old kmsg parser and create a new one
// enter the watch loop again
func (k *kernelLogWatcher) reviveMyself() {
	// if k.reviveCount >= reviveRetries {
	// 	klog.Errorf("Failed to revive kmsg parser after %d retries", reviveRetries)
	// 	return
	// }
	// klog.Infof("Reviving kmsg parser, attempt %d of %d", k.reviveCount, reviveRetries)
	klog.Infof("Reviving kmsg parser, attempt %d", k.reviveCount)
	if err := k.kmsgParser.Close(); err != nil {
		klog.Errorf("Failed to close kmsg parser: %v", err)
	}
	time.Sleep(reviveDuration)
	if err := k.SetKmsgParser(); err != nil {
		klog.Errorf("Failed to revive kmsg parser: %v", err)
		return
	}
	k.reviveCount++
	k.watchLoop()
}
