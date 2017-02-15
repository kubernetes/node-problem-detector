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
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	utilclock "code.cloudfoundry.org/clock"
	"github.com/golang/glog"
	"github.com/google/cadvisor/utils/tail"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/logwatchers/types"
	kerntypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
	"k8s.io/node-problem-detector/pkg/kernelmonitor/util"
)

type syslogWatcher struct {
	cfg        types.WatcherConfig
	reader     *bufio.Reader
	closer     io.Closer
	translator *translator
	logCh      chan *kerntypes.KernelLog
	uptime     time.Time
	tomb       *util.Tomb
	clock      utilclock.Clock
}

// NewSyslogWatcherOrDie creates a new kernel log watcher. The function panics
// when encounters an error.
func NewSyslogWatcherOrDie(cfg types.WatcherConfig) types.LogWatcher {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		glog.Fatalf("Failed to get system info: %v", err)
	}
	return &syslogWatcher{
		cfg:        cfg,
		translator: newTranslatorOrDie(cfg.PluginConfig),
		uptime:     time.Now().Add(time.Duration(-info.Uptime * int64(time.Second))),
		tomb:       util.NewTomb(),
		// A capacity 1000 buffer should be enough
		logCh: make(chan *kerntypes.KernelLog, 1000),
		clock: utilclock.NewClock(),
	}
}

// Make sure NewSyslogWathcer is types.WatcherCreateFunc.
var _ types.WatcherCreateFunc = NewSyslogWatcherOrDie

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
	lookback, err := time.ParseDuration(s.cfg.Lookback)
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
		line = buffer.String()
		buffer.Reset()
		log, err := s.translator.translate(strings.TrimSuffix(line, "\n"))
		if err != nil {
			glog.Warningf("Unable to parse line: %q, %v", line, err)
			continue
		}
		// If the log is older than look back duration or system boot time, discard it.
		if s.clock.Since(log.Timestamp) > lookback || log.Timestamp.Before(s.uptime) {
			continue
		}
		s.logCh <- log
	}
}

// getLogReader returns log reader for syslog log. Note that getLogReader doesn't look back
// to the rolled out logs.
func getLogReader(path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, fmt.Errorf("unexpected empty log path")
	}
	// To handle log rotation, tail will not report error immediately if
	// the file doesn't exist. So we check file existence first.
	// This could go wrong during mid-rotation. It should recover after
	// several restart when the log file is created again. The chance
	// is slim but we should still fix this in the future.
	// TODO(random-liu): Handle log missing during rotation.
	_, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat the file %q: %v", path, err)
	}
	tail, err := tail.NewTail(path)
	if err != nil {
		return nil, fmt.Errorf("failed to tail the file %q: %v", path, err)
	}
	return tail, nil
}
