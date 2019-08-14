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

	"k8s.io/node-problem-detector/pkg/util/metrics"
	"k8s.io/node-problem-detector/test/e2e/lib/gce"
	"k8s.io/node-problem-detector/test/e2e/lib/npd"

	"github.com/onsi/ginkgo"
	"github.com/pborman/uuid"
)

var _ = ginkgo.Describe("NPD should export Prometheus metrics.", func() {
	var instance gce.Instance

	ginkgo.BeforeEach(func() {
		var err error
		// TODO(xueweiz): Creating instance for each test case is slow. We should either reuse the instance
		// between tests, or have a way to run these tests in parallel.
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
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Unable to create test instance: %v", err))
		}

		err = npd.SetupNPD(instance, *npdBuildTar)
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Unable to setup NPD: %v", err))
		}
	})

	ginkgo.Context("On a clean node", func() {

		ginkgo.It("NPD should export host_uptime metric", func() {
			err := npd.WaitForNPD(instance, []string{"host_uptime"}, 120)
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Expect NPD to become ready in 120s, but hit error: %v", err))
			}

			gotMetrics, err := npd.FetchNPDMetrics(instance)
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Error fetching NPD metrics: %v", err))
			}
			_, err = metrics.GetFloat64Metric(gotMetrics, "host_uptime", map[string]string{}, false)
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Failed to find uptime metric: %v.\nHere is all NPD exported metrics: %v",
					err, gotMetrics))
			}
		})
	})

	ginkgo.AfterEach(func() {
		defer func() {
			err := instance.DeleteInstance()
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Failed to clean up the test VM: %v", err))
			}
		}()

		artifactSubDir := ""
		if *artifactsDir != "" {
			testText := ginkgo.CurrentGinkgoTestDescription().FullTestText
			testSubdirName := strings.Replace(testText, " ", "_", -1)

			artifactSubDir = path.Join(*artifactsDir, testSubdirName)
			err := os.MkdirAll(artifactSubDir, os.ModeDir|0644)
			if err != nil {
				fmt.Printf("Failed to create sub-directory to hold test artiface for test %s at %s\n",
					testText, artifactSubDir)
				return
			}
		}

		errs := npd.SaveTestArtifacts(instance, artifactSubDir)
		if len(errs) != 0 {
			fmt.Printf("Error storing debugging data to test artifacts: %v", errs)
		}
	})
})
