// +build !disable_system_log_monitor

/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/exporterplugins"
	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/problemdaemonplugins"
	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/exporters"
	"k8s.io/node-problem-detector/pkg/types"
)

const (
	fakeConfigFilePattern = `
{
	"plugin": "filelog",
	"pluginConfig": {
		"timestamp": "^time=\"(\\S*)\"",
		"message": "msg=\"([^\n]*)\"",
		"timestampFormat": "2006-01-02T15:04:05.999999999-07:00"
	},
	"logPath": "%s",
	"lookback": "5m",
	"bufferSize": 10,
	"source": "containerd",
	"conditions": [],
	"rules": [
		{
			"type": "temporary",
			"reason": "MissingPigz",
			"pattern": "unpigz not found.*"
		},
		{
			"type": "temporary",
			"reason": "IncompatibleContainer",
			"pattern": ".*CreateComputeSystem.*"
		}
	]
}
	`
)

func init() {
	exporters.Register("nil", types.ExporterHandler{
		CreateExporterOrDie: func(types.CommandLineOptions) types.Exporter {
			return &nullExporter{}
		},
	})
}

type nullExporter struct {
}

func (ne *nullExporter) ExportProblems(*types.Status) {
}

func TestNPDMain(t *testing.T) {
	npdo, cleanup := setupNPD(t)
	defer cleanup()

	termCh := make(chan error, 2)
	termCh <- errors.New("close")
	defer close(termCh)

	if err := npdMain(npdo, termCh); err != nil {
		t.Errorf("termination signal should not return error got, %v", err)
	}
}

func writeTempFile(t *testing.T, ext string, contents string) (string, error) {
	f, err := ioutil.TempFile("", "*."+ext)
	if err != nil {
		return "", fmt.Errorf("cannot create temp file, %v", err)
	}

	fileName := f.Name()

	if err := ioutil.WriteFile(fileName, []byte(contents), 0644); err != nil {
		os.Remove(fileName)
		return "", fmt.Errorf("cannot write config to temp file %s, %v", fileName, err)
	}

	return fileName, nil
}

func setupNPD(t *testing.T) (*options.NodeProblemDetectorOptions, func()) {
	fakeLogFileName, err := writeTempFile(t, "log", "")
	if err != nil {
		os.Remove(fakeLogFileName)
		t.Fatalf("cannot create temp config file, %v", err)
	}

	fakeConfigFileContents := fmt.Sprintf(fakeConfigFilePattern, strings.ReplaceAll(fakeLogFileName, "\\", "\\\\"))

	fakeConfigFileName, err := writeTempFile(t, "json", fakeConfigFileContents)
	if err != nil {
		os.Remove(fakeLogFileName)
		os.Remove(fakeConfigFileName)
		t.Fatalf("cannot create temp config file, %v", err)
	}

	return &options.NodeProblemDetectorOptions{
			MonitorConfigPaths: map[types.ProblemDaemonType]*[]string{
				"system-log-monitor": {
					fakeConfigFileName,
				},
			},
		}, func() {
			os.Remove(fakeLogFileName)
			os.Remove(fakeConfigFileName)
		}
}
