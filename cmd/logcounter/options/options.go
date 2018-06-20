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

package options

import (
	"flag"

	"github.com/spf13/pflag"
)

func NewLogCounterOptions() *LogCounterOptions {
	return &LogCounterOptions{}
}

// LogCounterOptions contains frequent event detector command line and application options.
type LogCounterOptions struct {
	// command line options. See flag descriptions for the description
	Lookback string
	Pattern  string
	Count    int
}

// AddFlags adds log counter command line options to pflag.
func (fedo *LogCounterOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&fedo.Lookback, "lookback", "", "The time log watcher looks up")
	fs.StringVar(&fedo.Pattern, "pattern", "",
		"The regular expression to match the problem in log. The pattern must match to the end of the line.")
	fs.IntVar(&fedo.Count, "count", 1,
		"The number of times the pattern must be found to trigger the condition")
}

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}
