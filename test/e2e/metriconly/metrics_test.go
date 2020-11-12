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

package e2e_metric_only

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"k8s.io/node-problem-detector/pkg/util/metrics"
	"k8s.io/node-problem-detector/test/e2e/lib/gce"
	"k8s.io/node-problem-detector/test/e2e/lib/npd"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
)

var _ = ginkgo.Describe("NPD should export Prometheus metrics.", func() {
	var instance gce.Instance

	ginkgo.BeforeEach(func() {
		var err error
		// TODO(xueweiz): Creating instance for each test case is slow. We should either reuse the instance
		// between tests, or have a way to run these tests in parallel.
		if *imageFamily != "" && *image == "" {
			gceImage, err := computeService.Images.GetFromFamily(*imageProject, *imageFamily).Do()
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Unable to get image from family %s at project %s: %v",
					*imageFamily, *imageProject, err))
			}
			*image = gceImage.Name
			fmt.Printf("Using image %s from image family %s at project %s\n", *image, *imageFamily, *imageProject)
		}
		instance, err = gce.CreateInstance(
			gce.Instance{
				Name:           "npd-metrics-" + *image + "-" + uuid.NewUUID().String()[:8],
				Zone:           *zone,
				Project:        *project,
				SshKey:         *sshKey,
				SshUser:        *sshUser,
				ComputeService: computeService,
			},
			*image,
			*imageProject)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Unable to create test instance: %v", err))

		err = npd.SetupNPD(instance, *npdBuildTar)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Unable to setup NPD: %v", err))
	})

	ginkgo.Context("On a clean node", func() {

		ginkgo.It("NPD should export cpu/disk/host/memory metric", func() {
			err := npd.WaitForNPD(instance, []string{"host_uptime"}, 120)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Expect NPD to become ready in 120s, but hit error: %v", err))

			gotMetrics, err := npd.FetchNPDMetrics(instance)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error fetching NPD metrics: %v", err))

			assertMetricExist(gotMetrics, "cpu_runnable_task_count", map[string]string{}, true)
			assertMetricExist(gotMetrics, "cpu_usage_time", map[string]string{}, false)
			assertMetricExist(gotMetrics, "cpu_load_1m", map[string]string{}, false)
			assertMetricExist(gotMetrics, "cpu_load_5m", map[string]string{}, false)
			assertMetricExist(gotMetrics, "cpu_load_15m", map[string]string{}, false)
			assertMetricExist(gotMetrics, "disk_operation_count", map[string]string{}, false)
			assertMetricExist(gotMetrics, "disk_merged_operation_count", map[string]string{}, false)
			assertMetricExist(gotMetrics, "disk_operation_bytes_count", map[string]string{}, false)
			assertMetricExist(gotMetrics, "disk_operation_time", map[string]string{}, false)
			assertMetricExist(gotMetrics, "disk_bytes_used", map[string]string{}, false)
			assertMetricExist(gotMetrics, "disk_io_time", map[string]string{}, false)
			assertMetricExist(gotMetrics, "disk_weighted_io", map[string]string{}, false)
			assertMetricExist(gotMetrics, "memory_bytes_used", map[string]string{}, false)
			assertMetricExist(gotMetrics, "memory_anonymous_used", map[string]string{}, false)
			assertMetricExist(gotMetrics, "memory_page_cache_used", map[string]string{}, false)
			assertMetricExist(gotMetrics, "memory_unevictable_used", map[string]string{}, true)
			assertMetricExist(gotMetrics, "memory_dirty_used", map[string]string{}, false)
			assertMetricExist(gotMetrics, "host_uptime", map[string]string{}, false)
		})

		ginkgo.It("NPD should not report any problem", func() {
			err := npd.WaitForNPD(instance, []string{"problem_gauge"}, 120)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Expect NPD to become ready in 120s, but hit error: %v", err))

			assertMetricValueInBound(instance,
				"problem_gauge", map[string]string{"reason": "DockerHung", "type": "KernelDeadlock"},
				0.0, 0.0)
			assertMetricValueInBound(instance,
				"problem_counter", map[string]string{"reason": "DockerHung"},
				0.0, 0.0)
			assertMetricValueInBound(instance,
				"problem_counter", map[string]string{"reason": "FilesystemIsReadOnly"},
				0.0, 0.0)
			assertMetricValueInBound(instance,
				"problem_counter", map[string]string{"reason": "KernelOops"},
				0.0, 0.0)
			assertMetricValueInBound(instance,
				"problem_counter", map[string]string{"reason": "OOMKilling"},
				0.0, 0.0)
		})
	})

	ginkgo.Context("When ext4 filesystem error happens", func() {

		ginkgo.BeforeEach(func() {
			err := npd.WaitForNPD(instance, []string{"problem_gauge"}, 120)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Expect NPD to become ready in 120s, but hit error: %v", err))
			// This will trigger a ext4 error on the boot disk, causing the boot disk mounted as read-only and systemd-journald crashing.
			instance.RunCommandOrFail("sudo /home/kubernetes/bin/problem-maker --problem Ext4FilesystemError")
		})

		ginkgo.It("NPD should update problem_counter{reason:Ext4Error} and problem_gauge{type:ReadonlyFilesystem}", func() {
			time.Sleep(5 * time.Second)
			assertMetricValueInBound(instance,
				"problem_counter", map[string]string{"reason": "Ext4Error"},
				1.0, 2.0)
			assertMetricValueInBound(instance,
				"problem_gauge", map[string]string{"reason": "FilesystemIsReadOnly", "type": "ReadonlyFilesystem"},
				1.0, 1.0)
		})

		ginkgo.It("NPD should remain healthy", func() {
			npdStates := instance.RunCommandOrFail("sudo systemctl show node-problem-detector -p ActiveState -p SubState")
			Expect(npdStates.Stdout).To(ContainSubstring("ActiveState=active"), "NPD is no longer active: %v", npdStates)
			Expect(npdStates.Stdout).To(ContainSubstring("SubState=running"), "NPD is no longer running: %v", npdStates)
		})
	})

	ginkgo.Context("When OOM kills and docker hung happen", func() {

		ginkgo.BeforeEach(func() {
			err := npd.WaitForNPD(instance, []string{"problem_gauge"}, 120)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Expect NPD to become ready in 120s, but hit error: %v", err))
			instance.RunCommandOrFail("sudo /home/kubernetes/bin/problem-maker --problem OOMKill")
			instance.RunCommandOrFail("sudo /home/kubernetes/bin/problem-maker --problem DockerHung")
		})

		ginkgo.It("NPD should update problem_counter and problem_gauge", func() {
			time.Sleep(5 * time.Second)
			assertMetricValueInBound(instance,
				"problem_counter", map[string]string{"reason": "DockerHung"},
				1.0, 1.0)
			assertMetricValueInBound(instance,
				"problem_counter", map[string]string{"reason": "TaskHung"},
				1.0, 1.0)
			assertMetricValueInBound(instance,
				"problem_gauge", map[string]string{"reason": "DockerHung", "type": "KernelDeadlock"},
				1.0, 1.0)
			assertMetricValueInBound(instance,
				"problem_counter", map[string]string{"reason": "OOMKilling"},
				1.0, 1.0)
		})
	})

	ginkgo.AfterEach(func() {
		defer func() {
			err := instance.DeleteInstance()
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to clena up the test VM: %v", err))
		}()

		artifactSubDir := ""
		if *artifactsDir != "" {
			testText := ginkgo.CurrentGinkgoTestDescription().FullTestText
			testSubdirName := strings.Replace(testText, " ", "_", -1)

			artifactSubDir = path.Join(*artifactsDir, testSubdirName)
			err := os.MkdirAll(artifactSubDir, os.ModeDir|0755)
			if err != nil {
				fmt.Printf("Failed to create sub-directory to hold test artiface for test %s at %s\n",
					testText, artifactSubDir)
				return
			}
		}

		errs := npd.SaveTestArtifacts(instance, artifactSubDir, config.GinkgoConfig.ParallelNode)
		if len(errs) != 0 {
			fmt.Printf("Error storing debugging data to test artifacts: %v", errs)
		}
	})
})

func assertMetricExist(metricList []metrics.Float64MetricRepresentation, metricName string, labels map[string]string, strictLabelMatching bool) {
	_, err := metrics.GetFloat64Metric(metricList, metricName, labels, strictLabelMatching)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to find metric %q: %v.\nHere is all NPD exported metrics: %v", metricName, err, metricList))
}

func assertMetricValueInBound(instance gce.Instance, metricName string, labels map[string]string, lowBound float64, highBound float64) {
	value, err := npd.FetchNPDMetric(instance, metricName, labels)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to find %s metric with label %v: %v", metricName, labels, err))
	}
	Expect(value).Should(BeNumerically(">=", lowBound),
		"Got value for metric %s with label %v: %v, expect at least %v.", metricName, labels, value, lowBound)
	Expect(value).Should(BeNumerically("<=", highBound),
		"Got value for metric %s with label %v: %v, expect at most %v.", metricName, labels, value, highBound)
}
