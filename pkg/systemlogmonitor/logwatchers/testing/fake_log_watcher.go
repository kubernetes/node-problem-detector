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

package testing

import (
	"sync"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
)

// FakeLogWatcher is a fake mock of log watcher.
type FakeLogWatcher struct {
	sync.Mutex
	buf chan *logtypes.Log
	err error
}

var _ types.LogWatcher = &FakeLogWatcher{}

func NewFakeLogWatcher(bufferSize int) *FakeLogWatcher {
	return &FakeLogWatcher{buf: make(chan *logtypes.Log, bufferSize)}
}

// InjectLog injects a fake log into the watch channel
func (f *FakeLogWatcher) InjectLog(log *logtypes.Log) {
	f.buf <- log
}

// InjectError injects an error of Watch function.
func (f *FakeLogWatcher) InjectError(err error) {
	f.Lock()
	defer f.Unlock()
	f.err = err
}

// Watch is the fake watch function.
func (f *FakeLogWatcher) Watch() (<-chan *logtypes.Log, error) {
	return f.buf, f.err
}

// Stop is the fake stop function.
func (f *FakeLogWatcher) Stop() {
	close(f.buf)
}
