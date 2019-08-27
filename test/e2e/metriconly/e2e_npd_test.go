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
	"flag"
	"fmt"
	"os"
	"path"
	"testing"

	"k8s.io/node-problem-detector/test/e2e/lib/gce"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	compute "google.golang.org/api/compute/v1"
)

const junitFileName = "junit.xml"

var zone = flag.String("zone", "", "gce zone the hosts live in")
var project = flag.String("project", "", "gce project the hosts live in")
var image = flag.String("image", "", "image to test")
var imageProject = flag.String("image-project", "", "gce project of the OS image")
var sshKey = flag.String("ssh-key", "", "path to ssh private key.")
var sshUser = flag.String("ssh-user", "", "use predefined user for ssh.")
var npdBuildTar = flag.String("npd-build-tar", "", "tarball containing NPD to be tested.")
var artifactsDir = flag.String("artifacts-dir", "", "local directory to save test artifacts into.")

var computeService *compute.Service

func TestNPD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	if *artifactsDir != "" {
		_, err := os.Stat(*artifactsDir)
		if err != nil && os.IsNotExist(err) {
			os.MkdirAll(*artifactsDir, os.ModeDir|0755)
		}
	}

	// The junit formatted result output is for showing test results on testgrid.
	junitReporter := reporters.NewJUnitReporter(path.Join(*artifactsDir, junitFileName))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "NPD Metric-only Suite", []ginkgo.Reporter{junitReporter})
}

func TestMain(m *testing.M) {
	flag.Parse()

	var err error
	computeService, err = gce.GetComputeClient()
	if err != nil {
		panic(fmt.Sprintf("Unable to create gcloud compute service using defaults. Make sure you are authenticated. %v", err))
	}

	os.Exit(m.Run())
}
