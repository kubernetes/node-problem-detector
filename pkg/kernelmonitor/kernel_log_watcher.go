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

package kernelmonitor

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"time"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/translator"
	"k8s.io/node-problem-detector/pkg/kernelmonitor/types"
	"k8s.io/node-problem-detector/pkg/kernelmonitor/util"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/golang/glog"
	"github.com/google/cadvisor/utils/tail"
	utilclock "github.com/pivotal-golang/clock"
)

const (
	defaultKernelLogPath = "/var/log/kern.log"
)

// WatcherConfig is the configuration of kernel log watcher.
type WatcherConfig struct {
	// KernelLogPath is the path to the kernel log
	KernelLogPath string `json:"logPath, omitempty"`
	// StartPattern is the pattern of the start line
	StartPattern string `json:"startPattern, omitempty"`
	// Lookback is the time kernel watcher looks up
	Lookback string `json:"lookback, omitempty"`
}

// KernelLogWatcher watches and translates the kernel log. Once there is new log line,
// it will translate and report the log.
type KernelLogWatcher interface {
	// Watch starts the kernel log watcher and returns a watch channel.
	Watch() (<-chan *types.KernelLog, error)
	// Stop stops the kernel log watcher.
	Stop()
}

type kernelLogWatcher struct {
	// trans is the translator translates the log into internal format.
	trans  translator.Translator
	cfg    WatcherConfig
	reader *bufio.Reader
	logCh  chan *types.KernelLog
	tomb   *util.Tomb
	clock  utilclock.Clock
}

// NewKernelLogWatcher creates a new kernel log watcher.
func NewKernelLogWatcher(cfg WatcherConfig) KernelLogWatcher {
	return &kernelLogWatcher{
		trans: translator.NewDefaultTranslator(),
		cfg:   cfg,
		tomb:  util.NewTomb(),
		// A capacity 1000 buffer should be enough
		logCh: make(chan *types.KernelLog, 1000),
		clock: utilclock.NewClock(),
	}
}

func (k *kernelLogWatcher) Watch() (<-chan *types.KernelLog, error) {
	path := defaultKernelLogPath
	if k.cfg.KernelLogPath != "" {
		path = k.cfg.KernelLogPath
	}
	// NOTE(random-liu): This is a hack. KernelMonitor doesn't support some OS distros e.g. GCI. Ideally,
	// KernelMonitor should only run on nodes with supported OS distro. However, NodeProblemDetector is
	// running as DaemonSet, it has to be deployed on each node (There is no node affinity support for
	// DaemonSet now #22205). If some nodes have unsupported OS distro e.g. the OS distro of master node
	// in gke/gce is GCI, KernelMonitor will keep throwing out error, and NodeProblemDetector will be
	// restarted again and again.
	// To avoid this, we decide to add this temporarily hack. When KernelMonitor can't find the kernel
	// log file, it will print a log and then return nil channel and no error. Since nil channel will
	// always be blocked, the NodeProblemDetector will block forever.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		glog.Infof("kernel log %q is not found, kernel monitor doesn't support the os distro", path)
		return nil, nil
	}
	// TODO(random-liu): Rate limit tail file.
	// Notice that, kernel log watcher doesn't look back to the rolled out logs.
	reader, err := getLogReader(path)
	if err != nil {
		return nil, err
	}
	k.reader = bufio.NewReader(reader)
	glog.Info("Start watching kernel log")
	go k.watchLoop()
	return k.logCh, nil
}

func (k *kernelLogWatcher) Stop() {
	k.tomb.Stop()
}

// watchLoop is the main watch loop of kernel log watcher.
func (k *kernelLogWatcher) watchLoop() {
	defer func() {
		close(k.logCh)
		k.tomb.Done()
	}()
	lookback, err := parseDuration(k.cfg.Lookback)
	if err != nil {
		glog.Fatalf("failed to parse duration %q: %v", k.cfg.Lookback, err)
	}
	var buffer bytes.Buffer
	for {

		select {
		case <-k.tomb.Stopping():
			glog.Infof("Stop watching kernel log")
			return
		default:
		}

		line, err := k.reader.ReadString('\n')
		if err != nil && err != io.EOF {
			glog.Errorf("exiting kernel log watch with error: %v", err)
			return
		}
		if line == "" {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if err == nil {
			buffer.WriteString(line)
			// trime `\n`
			line = strings.TrimRight(buffer.String(), "\n")
			buffer.Reset()
			log, err := k.trans.Translate(line)
			if err != nil {
				glog.Infof("Unable to parse line: %q, %v", line, err)
				continue
			}
			// If the log is older than look back duration, discard it.
			if k.clock.Since(log.Timestamp) > lookback {
				continue
			}
			k.logCh <- log
		} else { // err == io.EOF
			buffer.WriteString(line)
		}
	}
}

// getLogReader gets a kernel log reader.
func getLogReader(path string) (io.Reader, error) {
	reader, err := tryJournal()
	if err == nil {
		return reader, nil
	}
	reader, err = tryLogFile(path)
	if err == nil {
		return reader, nil
	}
	return nil, err
}

func tryJournal() (io.Reader, error) {
	r, err := sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
		NumFromTail: uint64(0),
		Matches: []sdjournal.Match{
			{
				Field: sdjournal.SD_JOURNAL_FIELD_TRANSPORT,
				Value: "kernel",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Error opening journal: %v", err)
	}
	if r == nil {
		return nil, fmt.Errorf("Got a nil reader")
	}
	glog.Info("Kernel log watcher use journal")
	return r, nil
}

func tryLogFile(path string) (io.Reader, error) {
	tail, err := tail.NewTail(path)
	if err != nil {
		return nil, err
	}
	glog.Infof("Kernel log watcher use log file: %s", path)
	return tail, nil
}

func parseDuration(s string) (time.Duration, error) {
	// If the duration is not configured, just return 0 by default
	if s == "" {
		return 0, nil
	}
	return time.ParseDuration(s)
}
