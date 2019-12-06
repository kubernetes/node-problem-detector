/*
Copyright 2019 The Kubernetes Authors.

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

package npd

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"k8s.io/node-problem-detector/pkg/util/metrics"
	"k8s.io/node-problem-detector/test/e2e/lib/gce"

	"github.com/avast/retry-go"
)

// SetupNPD installs NPD from the test tarball onto the provided GCE instance.
//
// Here is how it works:
// 1. SetupNPD will SCP the NPD build tarball onto the VM.
// 2. SetupNPD will extract the tarball in the VM, to expose the test/e2e-install.sh on the VM.
// 3. SetupNPD will then call the e2e-install.sh script, and feed the NPD build tarball as input.
// 4. Finally, the e2e-install.sh script will do the heavy lifting of installing NPD (setting up
//    binary/config directories, setting up systemd config file, etc).
func SetupNPD(ins gce.Instance, npdBuildTar string) error {
	tmpDirCmd := ins.RunCommand("mktemp -d")
	if tmpDirCmd.SSHError != nil || tmpDirCmd.Code != 0 {
		return fmt.Errorf("error creating temporary directory to hold NPD tarball: %v", tmpDirCmd)
	}

	tmpDir := strings.TrimSuffix(tmpDirCmd.Stdout, "\n")
	npdTarVMPath := tmpDir + "/npd.tar.gz"
	npdExtractDir := tmpDir + "/npd"

	err := ins.PushFile(npdBuildTar, npdTarVMPath)
	if err != nil {
		return fmt.Errorf("error pushing local NPD build tarball %s to VM at %s: %v", npdBuildTar, npdTarVMPath, err)
	}

	mkdirCmd := ins.RunCommand(fmt.Sprintf("mkdir -p %s", npdExtractDir))
	if mkdirCmd.SSHError != nil || mkdirCmd.Code != 0 {
		return fmt.Errorf("error creating directory to extract NPD tarball into: %v", mkdirCmd)
	}

	extractCmd := ins.RunCommand(fmt.Sprintf("tar -xf %s --directory %s", npdTarVMPath, npdExtractDir))
	if extractCmd.SSHError != nil || extractCmd.Code != 0 {
		return fmt.Errorf("error extracting NPD build tarball: %v", extractCmd)
	}

	installCmd := ins.RunCommand(fmt.Sprintf("sudo bash %s/test/e2e-install.sh -t %s install", npdExtractDir, npdTarVMPath))
	if installCmd.SSHError != nil || installCmd.Code != 0 {
		return fmt.Errorf("error installing NPD: %v", installCmd)
	}

	return nil
}

// FetchNPDMetrics fetches and parses metrics reported by NPD on the provided GCE instance.
func FetchNPDMetrics(ins gce.Instance) ([]metrics.Float64MetricRepresentation, error) {
	var npdMetrics []metrics.Float64MetricRepresentation
	var err error

	curlCmd := ins.RunCommand("curl http://localhost:20257/metrics")
	if curlCmd.SSHError != nil || curlCmd.Code != 0 {
		return npdMetrics, fmt.Errorf("error fetching NPD metrics: %v", curlCmd)
	}

	npdMetrics, err = metrics.ParsePrometheusMetrics(curlCmd.Stdout)
	if err != nil {
		return npdMetrics, fmt.Errorf("error parsing NPD metrics: %v", err)
	}

	return npdMetrics, nil
}

// FetchNPDMetric fetches and parses a specific metric reported by NPD on the provided GCE instance.
func FetchNPDMetric(ins gce.Instance, metricName string, labels map[string]string) (float64, error) {
	gotMetrics, err := FetchNPDMetrics(ins)
	if err != nil {
		return 0.0, err
	}
	metric, err := metrics.GetFloat64Metric(gotMetrics, metricName, labels, true)
	if err != nil {
		return 0.0, fmt.Errorf("Failed to find %s metric with label %v: %v.\nHere is all NPD exported metrics: %v",
			metricName, labels, err, gotMetrics)
	}
	return metric.Value, nil
}

// WaitForNPD waits for NPD to become ready by waiting for expected metrics.
func WaitForNPD(ins gce.Instance, metricNames []string, timeoutSeconds uint) error {
	verifyMetricExist := func() error {
		gotMetrics, err := FetchNPDMetrics(ins)
		if err != nil {
			return fmt.Errorf("Error fetching NPD metrics: %v", err)
		}
		for _, metricName := range metricNames {
			_, err = metrics.GetFloat64Metric(gotMetrics, metricName, map[string]string{}, false)
			if err != nil {
				return fmt.Errorf("Failed to find metric %s: %v.\nHere is all NPD exported metrics: %v",
					metricName, err, gotMetrics)
			}
		}
		return nil
	}

	// Wait for NPD to be ready for a maximum of 120 seconds.
	return retry.Do(verifyMetricExist,
		retry.Delay(10*time.Second),
		retry.Attempts(timeoutSeconds/10),
		retry.DelayType(retry.FixedDelay))
}

// SaveTestArtifacts saves debugging data from NPD.
func SaveTestArtifacts(ins gce.Instance, artifactDirectory string, testID int) []error {
	var errs []error

	if err := saveCommandResultAsArtifact(ins, artifactDirectory, testID,
		"curl http://localhost:20257/metrics", "node-problem-detector-metrics"); err != nil {
		errs = append(errs, err)
	}
	if err := saveCommandResultAsArtifact(ins, artifactDirectory, testID,
		"sudo journalctl -u node-problem-detector.service", "node-problem-detector"); err != nil {
		errs = append(errs, err)
	}
	if err := saveCommandResultAsArtifact(ins, artifactDirectory, testID,
		"sudo journalctl -k", "kernel-logs"); err != nil {
		errs = append(errs, err)
	}

	return errs
}

func saveCommandResultAsArtifact(ins gce.Instance, artifactDirectory string, testID int, command string, artifactPrefix string) error {
	artifactPath := path.Join(artifactDirectory, fmt.Sprintf("%v-%02d.txt", artifactPrefix, testID))
	result := ins.RunCommand(command)
	if result.SSHError != nil || result.Code != 0 {
		return fmt.Errorf("Error running command: %v\n", result)
	}
	if err := ioutil.WriteFile(artifactPath, []byte(result.Stdout), 0644); err != nil {
		return fmt.Errorf("Error writing artifact to %v: %v\n", artifactPath, err)
	}
	return nil
}
