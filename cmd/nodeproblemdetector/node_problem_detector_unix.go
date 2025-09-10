//go:build unix

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

	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/cmd/options"
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
	if err := pflag.CommandLine.MarkHidden("vmodule"); err != nil {
		klog.Fatalf("Failed to mark hidden flag 'vmodule': %v", err)
	}
	if err := pflag.CommandLine.MarkHidden("logtostderr"); err != nil {
		klog.Fatalf("Failed to mark hidden flag 'logtostderr': %v", err)
	}

	npdo := options.NewNodeProblemDetectorOptions()
	if err := npdo.AddFlags(pflag.CommandLine); err != nil {
		klog.Fatalf("Failed to add flags: %v", err)
	}

	pflag.Parse()
	if err := npdMain(context.Background(), npdo); err != nil {
		klog.Fatalf("Problem detector failed with error: %v", err)
	}
}
