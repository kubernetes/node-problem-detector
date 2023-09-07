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
	"fmt"
	"regexp"
	"time"

	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"

	"k8s.io/klog/v2"
)

// translator translates log line into internal log type based on user defined
// regular expression.
type translator struct {
	timestampRegexp *regexp.Regexp
	messageRegexp   *regexp.Regexp
	timestampFormat string
}

const (
	// NOTE that we support submatch for both timestamp and message regular expressions. When
	// there are multiple matches returned by submatch, only **the last** is used.
	// timestampKey is the key of timestamp regular expression in the plugin configuration.
	timestampKey = "timestamp"
	// messageKey is the key of message regular expression in the plugin configuration.
	messageKey = "message"
	// timestampFormatKey is the key of timestamp format string in the plugin configuration.
	timestampFormatKey = "timestampFormat"
)

func newTranslatorOrDie(pluginConfig map[string]string) *translator {
	if err := validatePluginConfig(pluginConfig); err != nil {
		klog.Errorf("Failed to validate plugin configuration %+v: %v", pluginConfig, err)
	}
	return &translator{
		timestampRegexp: regexp.MustCompile(pluginConfig[timestampKey]),
		messageRegexp:   regexp.MustCompile(pluginConfig[messageKey]),
		timestampFormat: pluginConfig[timestampFormatKey],
	}
}

// translate translates the log line into internal type.
func (t *translator) translate(line string) (*logtypes.Log, error) {
	// Parse timestamp.
	matches := t.timestampRegexp.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no timestamp found in line %q with regular expression %v", line, t.timestampRegexp)
	}
	timestamp, err := time.ParseInLocation(t.timestampFormat, matches[len(matches)-1], time.Local)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp %q: %v", matches[len(matches)-1], err)
	}
	// Formalize the timestamp.
	timestamp = formalizeTimestamp(timestamp)
	// Parse message.
	matches = t.messageRegexp.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no message found in line %q with regular expression %v", line, t.messageRegexp)
	}
	message := matches[len(matches)-1]
	return &logtypes.Log{
		Timestamp: timestamp,
		Message:   message,
	}, nil
}

// validatePluginConfig validates whether the plugin configuration.
func validatePluginConfig(cfg map[string]string) error {
	if cfg[timestampKey] == "" {
		return fmt.Errorf("unexpected empty timestamp regular expression")
	}
	if cfg[messageKey] == "" {
		return fmt.Errorf("unexpected empty message regular expression")
	}
	if cfg[timestampFormatKey] == "" {
		return fmt.Errorf("unexpected empty timestamp format string")
	}
	return nil
}

// formalizeTimestamp formalizes the timestamp. We need this because some log doesn't contain full
// timestamp, e.g. filelog.
func formalizeTimestamp(t time.Time) time.Time {
	if t.Year() == 0 {
		t = t.AddDate(time.Now().Year(), 0, 0)
	}
	return t
}
