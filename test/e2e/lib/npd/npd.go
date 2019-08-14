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

const npdMetricsFilename = "node-problem-detector-metrics.txt"
const npdLogsFilename = "node-problem-detector.log"

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
func SaveTestArtifacts(ins gce.Instance, directory string) []error {
	var errs []error

	npdMetrics := ins.RunCommand("curl http://localhost:20257/metrics")
	if npdMetrics.SSHError != nil || npdMetrics.Code != 0 {
		errs = append(errs, fmt.Errorf("Error fetching NPD metrics: %v\n", npdMetrics))
	} else {
		npdMetricsPath := path.Join(directory, npdMetricsFilename)
		err := ioutil.WriteFile(npdMetricsPath, []byte(npdMetrics.Stdout), 0644)
		if err != nil {
			errs = append(errs, fmt.Errorf("Error writing to %s: %v", npdMetricsPath, err))
		}
	}

	npdLog := ins.RunCommand("sudo journalctl -u node-problem-detector.service")
	if npdLog.SSHError != nil || npdLog.Code != 0 {
		errs = append(errs, fmt.Errorf("Error fetching NPD logs: %v\n", npdLog))
	} else {
		npdLogsPath := path.Join(directory, npdLogsFilename)
		err := ioutil.WriteFile(npdLogsPath, []byte(npdLog.Stdout), 0644)
		if err != nil {
			errs = append(errs, fmt.Errorf("Error writing to %s: %v", npdLogsPath, err))
		}
	}

	return errs
}
