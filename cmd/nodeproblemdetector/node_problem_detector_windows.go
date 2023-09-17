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
	"context"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/pflag"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"k8s.io/klog/v2"
	"k8s.io/node-problem-detector/cmd/options"
)

const (
	svcName             = "NodeProblemDetector"
	svcDescription      = "Identifies problems that likely disrupt the operation of Kubernetes workloads."
	svcCommandsAccepted = svc.AcceptStop | svc.AcceptShutdown
	appEventLogName     = svcName
	windowsEventLogID   = 1
)

var (
	elog debug.Log
)

func main() {
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	klogFlags.VisitAll(func(f *flag.Flag) {
		switch f.Name {
		case "v", "vmodule", "logtostderr":
			flag.CommandLine.Var(f.Value, f.Name, f.Usage)
		}
	})
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.MarkHidden("vmodule")
	pflag.CommandLine.MarkHidden("logtostderr")

	npdo := options.NewNodeProblemDetectorOptions()
	npdo.AddFlags(pflag.CommandLine)

	pflag.Parse()

	handler := &npdService{
		options: npdo,
	}

	runFunc := initializeRun()

	if err := runFunc(svcName, handler); err != nil {
		elog.Error(windowsEventLogID, err.Error())
	}
}

func isRunningAsWindowsService() bool {
	runningAsService, err := svc.IsWindowsService()
	if err != nil {
		klog.Errorf("cannot determine if running as Windows Service assuming standalone, %v", err)
		return false
	}
	return runningAsService
}

func setupLogging(runningAsService bool) {
	if runningAsService {
		var err error

		elog, err = eventlog.Open(appEventLogName)

		// If the event log is unavailable then at least print out to standard output.
		if err != nil {
			elog = debug.New(appEventLogName)
			elog.Info(windowsEventLogID, fmt.Sprintf("cannot connect to event log using standard out, %v", err))
		}
	} else {
		elog = debug.New(appEventLogName)
	}
}

func initializeRun() func(string, svc.Handler) error {
	runningAsService := isRunningAsWindowsService()

	setupLogging(runningAsService)

	if runningAsService {
		return svc.Run
	}

	return debug.Run
}

type npdService struct {
	sync.Mutex
	options *options.NodeProblemDetectorOptions
}

func (s *npdService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: svcCommandsAccepted}
	var appWG sync.WaitGroup
	var svcWG sync.WaitGroup

	options := s.options
	ctx, cancelFunc := context.WithCancel(context.Background())

	// NPD application goroutine.
	appWG.Add(1)
	go func() {
		defer appWG.Done()

		if err := npdMain(ctx, options); err != nil {
			elog.Warning(windowsEventLogID, err.Error())
		}

		changes <- svc.Status{State: svc.StopPending}
	}()

	// Windows service control goroutine.
	svcWG.Add(1)
	go func() {
		defer svcWG.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case c := <-r:
				switch c.Cmd {
				case svc.Interrogate:
					changes <- c.CurrentStatus
					// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
					time.Sleep(100 * time.Millisecond)
					changes <- c.CurrentStatus
				case svc.Stop, svc.Shutdown:
					elog.Info(windowsEventLogID, fmt.Sprintf("Stopping %s service, %v", svcName, c.Context))
					cancelFunc()
				case svc.Pause:
					elog.Info(windowsEventLogID, "ignoring pause command from Windows service control, not supported")
					changes <- svc.Status{State: svc.Paused, Accepts: svcCommandsAccepted}
				case svc.Continue:
					elog.Info(windowsEventLogID, "ignoring continue command from Windows service control, not supported")
					changes <- svc.Status{State: svc.Running, Accepts: svcCommandsAccepted}
				default:
					elog.Error(windowsEventLogID, fmt.Sprintf("unexpected control request #%d", c))
				}
			}
		}
	}()

	// Wait for the application go routine to die.
	appWG.Wait()

	// Wait for the service control loop to terminate.
	// Otherwise it's possible that the channel closures cause the application to panic.
	svcWG.Wait()

	// Send a signal to the Windows service control that the application has stopped.
	changes <- svc.Status{State: svc.Stopped, Accepts: svcCommandsAccepted}

	return false, uint32(0)
}
