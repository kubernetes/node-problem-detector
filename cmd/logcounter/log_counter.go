//go:build journald
// +build journald

/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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
	"flag"
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"k8s.io/klog/v2"
	"k8s.io/node-problem-detector/cmd/logcounter/options"
	"k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/logcounter"
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

	fedo := options.NewLogCounterOptions()
	fedo.AddFlags(pflag.CommandLine)
	pflag.Parse()

	counter, err := logcounter.NewJournaldLogCounter(fedo)
	if err != nil {
		fmt.Print(err)
		os.Exit(int(types.Unknown))
	}
	actual, err := counter.Count()
	if err != nil {
		fmt.Print(err)
		os.Exit(int(types.Unknown))
	}
	if actual >= fedo.Count {
		fmt.Printf("Found %d matching logs, which meets the threshold of %d\n", actual, fedo.Count)
		os.Exit(int(types.NonOK))
	}
	os.Exit(int(types.OK))
}
