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

package filelog

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"time"

	utilclock "code.cloudfoundry.org/clock"
	"github.com/golang/glog"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

type filelogWatcher struct {
	cfg        types.WatcherConfig
	reader     *bufio.Reader
	closer     io.Closer
	translator *translator
	logCh      chan *logtypes.Log
	startTime  time.Time
	tomb       *tomb.Tomb
	clock      utilclock.Clock
}

// NewSyslogWatcherOrDie creates a new log watcher. The function panics
// when encounters an error.
func NewSyslogWatcherOrDie(cfg types.WatcherConfig) types.LogWatcher {
	uptime, err := util.GetUptimeDuration()
	if err != nil {
		glog.Fatalf("failed to get uptime: %v", err)
	}
	startTime, err := util.GetStartTime(time.Now(), uptime, cfg.Lookback, cfg.Delay)
	if err != nil {
		glog.Fatalf("failed to get start time: %v", err)
	}

	return &filelogWatcher{
		cfg:        cfg,
		translator: newTranslatorOrDie(cfg.PluginConfig),
		startTime:  startTime,
		tomb:       tomb.NewTomb(),
		// A capacity 1000 buffer should be enough
		logCh: make(chan *logtypes.Log, 1000),
		clock: utilclock.NewClock(),
	}
}

// Make sure NewSyslogWatcher is types.WatcherCreateFunc.
var _ types.WatcherCreateFunc = NewSyslogWatcherOrDie

// Watch starts the filelog watcher.
func (s *filelogWatcher) Watch() (<-chan *logtypes.Log, error) {
	r, err := getLogReader(s.cfg.LogPath)
	if err != nil {
		return nil, err
	}
	s.reader = bufio.NewReader(r)
	s.closer = r
	glog.Info("Start watching filelog")
	go s.watchLoop()
	return s.logCh, nil
}

// Stop stops the filelog watcher.
func (s *filelogWatcher) Stop() {
	s.tomb.Stop()
}

// watchPollInterval is the interval filelog log watcher will
// poll for pod change after reading to the end.
const watchPollInterval = 500 * time.Millisecond

// watchLoop is the main watch loop of filelog watcher.
func (s *filelogWatcher) watchLoop() {
	defer func() {
		s.closer.Close()
		close(s.logCh)
		s.tomb.Done()
	}()
	var buffer bytes.Buffer
	for {
		select {
		case <-s.tomb.Stopping():
			glog.Infof("Stop watching filelog")
			return
		default:
		}

		line, err := s.reader.ReadString('\n')
		if err != nil && err != io.EOF {
			glog.Errorf("Exiting filelog watch with error: %v", err)
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
		// Discard messages before start time.
		if log.Timestamp.Before(s.startTime) {
			glog.V(5).Infof("Throwing away msg %q before start time: %v < %v", log.Message, log.Timestamp, s.startTime)
			continue
		}
		s.logCh <- log
	}
}
