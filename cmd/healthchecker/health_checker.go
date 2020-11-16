/*
Copyright 2020 The Kubernetes Authors All rights reserved.

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

	"k8s.io/node-problem-detector/cmd/healthchecker/options"
	"k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/healthchecker"
)

func main() {
	// Set glog flag so that it does not log to files.
	if err := flag.Set("logtostderr", "true"); err != nil {
		fmt.Printf("Failed to set logtostderr=true: %v", err)
		os.Exit(int(types.Unknown))
	}

	hco := options.NewHealthCheckerOptions()
	hco.AddFlags(pflag.CommandLine)
	pflag.Parse()
	hco.SetDefaults()
	if err := hco.IsValid(); err != nil {
		fmt.Println(err)
		os.Exit(int(types.Unknown))
	}

	hc, err := healthchecker.NewHealthChecker(hco)
	if err != nil {
		fmt.Println(err)
		os.Exit(int(types.Unknown))
	}
	healthy, err := hc.CheckHealth()
	if err != nil {
		fmt.Printf("error checking %v health: %v\n", hco.Component, err)
		os.Exit(int(types.Unknown))
	}
	if !healthy {
		fmt.Printf("%v:%v was found unhealthy; repair flag : %v\n", hco.Component, hco.SystemdService, hco.EnableRepair)
		os.Exit(int(types.NonOK))
	}
	os.Exit(int(types.OK))
}
