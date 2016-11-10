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
	"strings"
	"time"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/plugins"
	plugtypes "k8s.io/node-problem-detector/pkg/kernelmonitor/plugins/types"
	"k8s.io/node-problem-detector/pkg/kernelmonitor/types"
	"k8s.io/node-problem-detector/pkg/kernelmonitor/util"

	"github.com/golang/glog"
	utilclock "github.com/pivotal-golang/clock"
)

// WatcherConfig is the configuration of kernel log watcher.
type WatcherConfig struct {
	// Configuration of plugins.
	plugtypes.Config
	// StartPattern is the pattern of the start line
	StartPattern string `json:"startPattern, omitempty"`
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
	cfg    WatcherConfig
	plugin plugtypes.Plugin
	reader *bufio.Reader
	logCh  chan *types.KernelLog
	tomb   *util.Tomb
	clock  utilclock.Clock
}

// NewKernelLogWatcher creates a new kernel log watcher.
func NewKernelLogWatcher(cfg WatcherConfig) KernelLogWatcher {
	return &kernelLogWatcher{
		cfg:  cfg,
		tomb: util.NewTomb(),
		// A capacity 1000 buffer should be enough
		logCh: make(chan *types.KernelLog, 1000),
		clock: utilclock.NewClock(),
	}
}

func (k *kernelLogWatcher) Watch() (<-chan *types.KernelLog, error) {
	plugin, err := plugins.GetPlugin(k.cfg.Config)
	if err != nil {
		return nil, err
	}
	k.reader = bufio.NewReader(plugin)
	k.plugin = plugin
	glog.Info("Start watching kernel log")
	go k.watchLoop()
	return k.logCh, nil
}

func (k *kernelLogWatcher) Stop() {
	k.tomb.Stop()
}

// watchPollInterval is the interval kernel log watcher will
// poll for pod change after reading to the end.
const watchPollInterval = 1 * time.Second

// watchLoop is the main watch loop of kernel log watcher.
func (k *kernelLogWatcher) watchLoop() {
	defer func() {
		k.plugin.Close()
		close(k.logCh)
		k.tomb.Done()
	}()
	lookback, err := util.ParseDuration(k.cfg.Lookback)
	if err != nil {
		glog.Fatalf("Failed to parse duration %q: %v", k.cfg.Lookback, err)
	}
	glog.Info("Lookback:", lookback)
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
			glog.Errorf("Exiting kernel log watch with error: %v", err)
			return
		}
		buffer.WriteString(line)
		if err == io.EOF {
			time.Sleep(watchPollInterval)
			continue
		}
		// Trim tailing `\n`.
		line = strings.TrimRight(buffer.String(), "\n")
		buffer.Reset()
		log, err := k.plugin.Translate(line)
		if err != nil {
			glog.Infof("Unable to parse line: %q, %v", line, err)
			continue
		}
		// If the log is older than look back duration, discard it.
		if k.clock.Since(log.Timestamp) > lookback {
			continue
		}
		k.logCh <- log
	}
}
