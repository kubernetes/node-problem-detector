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

package syslog

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/golang/glog"
	utilclock "github.com/pivotal-golang/clock"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/logwatchers/types"
	kerntypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
	"k8s.io/node-problem-detector/pkg/kernelmonitor/util"
)

type syslogWatcher struct {
	cfg    types.WatcherConfig
	reader *bufio.Reader
	closer io.Closer
	logCh  chan *kerntypes.KernelLog
	tomb   *util.Tomb
	clock  utilclock.Clock
}

// NewSyslogWatcher creates a new kernel log watcher.
func NewSyslogWatcher(cfg types.WatcherConfig) types.LogWatcher {
	return &syslogWatcher{
		cfg:  cfg,
		tomb: util.NewTomb(),
		// A capacity 1000 buffer should be enough
		logCh: make(chan *kerntypes.KernelLog, 1000),
		clock: utilclock.NewClock(),
	}
}

// Make sure NewSyslogWathcer is types.WatcherCreateFunc.
var _ types.WatcherCreateFunc = NewSyslogWatcher

// Watch starts the syslog watcher.
func (s *syslogWatcher) Watch() (<-chan *kerntypes.KernelLog, error) {
	r, err := getLogReader(s.cfg.LogPath)
	if err != nil {
		return nil, err
	}
	s.reader = bufio.NewReader(r)
	s.closer = r
	glog.Info("Start watching syslog")
	go s.watchLoop()
	return s.logCh, nil
}

// Stop stops the syslog watcher.
func (s *syslogWatcher) Stop() {
	s.tomb.Stop()
}

// watchPollInterval is the interval syslog log watcher will
// poll for pod change after reading to the end.
const watchPollInterval = 500 * time.Millisecond

// watchLoop is the main watch loop of syslog watcher.
func (s *syslogWatcher) watchLoop() {
	defer func() {
		s.closer.Close()
		close(s.logCh)
		s.tomb.Done()
	}()
	lookback, err := util.ParseDuration(s.cfg.Lookback)
	if err != nil {
		glog.Fatalf("Failed to parse duration %q: %v", s.cfg.Lookback, err)
	}
	glog.Info("Lookback:", lookback)
	var buffer bytes.Buffer
	for {
		select {
		case <-s.tomb.Stopping():
			glog.Infof("Stop watching syslog")
			return
		default:
		}

		line, err := s.reader.ReadString('\n')
		if err != nil && err != io.EOF {
			glog.Errorf("Exiting syslog watch with error: %v", err)
			return
		}
		buffer.WriteString(line)
		if err == io.EOF {
			time.Sleep(watchPollInterval)
			continue
		}
		line = strings.TrimSpace(buffer.String())
		buffer.Reset()
		log, err := translate(line)
		if err != nil {
			glog.Infof("Unable to parse line: %q, %v", line, err)
			continue
		}
		// If the log is older than look back duration, discard it.
		if s.clock.Since(log.Timestamp) > lookback {
			continue
		}
		s.logCh <- log
	}
}
