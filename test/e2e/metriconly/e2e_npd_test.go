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
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"k8s.io/node-problem-detector/pkg/util/tomb"
	"k8s.io/node-problem-detector/test/e2e/lib/gce"
	"k8s.io/test-infra/boskos/client"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	compute "google.golang.org/api/compute/v1"
)

var zone = flag.String("zone", "", "gce zone the hosts live in")
var project = flag.String("project", "", "gce project the hosts live in")
var image = flag.String("image", "", "image to test")
var imageFamily = flag.String("image-family", "", "image family to pick up the test image. Ignored when -image is set.")
var imageProject = flag.String("image-project", "", "gce project of the OS image")
var jobName = flag.String("job-name", "", "name of the Prow job running the test")
var sshKey = flag.String("ssh-key", "", "path to ssh private key.")
var sshUser = flag.String("ssh-user", "", "use predefined user for ssh.")
var npdBuildTar = flag.String("npd-build-tar", "", "tarball containing NPD to be tested.")
var artifactsDir = flag.String("artifacts-dir", "", "local directory to save test artifacts into.")
var boskosProjectType = flag.String("boskos-project-type", "gce-project",
	"specifies which project type to select from Boskos.")
var boskosServerURL = flag.String("boskos-server-url", "http://boskos.test-pods.svc.cluster.local",
	"specifies Boskos server URL.")
var boskosWaitDuration = flag.Duration("boskos-wait-duration", 2*time.Minute,
	"Duration to wait before quitting getting Boskos resource.")

var computeService *compute.Service

// boskosClient helps renting project from Boskos, and is only initialized on Ginkgo node 1.
var boskosClient *client.Client

// boskosRenewingTomb stops the goroutine keep renewing the Boskos resources.
var boskosRenewingTomb *tomb.Tomb

// SynchronizedBeforeSuite and SynchronizedAfterSuite help manages singleton resource (a Boskos project) across Ginkgo nodes.
var _ = ginkgo.SynchronizedBeforeSuite(rentBoskosProjectIfNeededOnNode1, acceptBoskosProjectIfNeededFromNode1)
var _ = ginkgo.SynchronizedAfterSuite(func() {}, releaseBoskosResourcesOnNode1)

func TestNPD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var err error
	computeService, err = gce.GetComputeClient()
	if err != nil {
		panic(fmt.Sprintf("Unable to create gcloud compute service using defaults. Make sure you are authenticated. %v", err))
	}

	if *artifactsDir != "" {
		_, err := os.Stat(*artifactsDir)
		if err != nil && os.IsNotExist(err) {
			os.MkdirAll(*artifactsDir, os.ModeDir|0755)
		}
	}

	// The junit formatted result output is for showing test results on testgrid.
	junitReporter := reporters.NewJUnitReporter(path.Join(*artifactsDir, fmt.Sprintf("junit-%02d.xml", config.GinkgoConfig.ParallelNode)))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "NPD Metric-only Suite", []ginkgo.Reporter{junitReporter})
}

// rentBoskosProjectIfNeededOnNode1 rents a GCP project from Boskos if no GCP project is specified.
//
// rentBoskosProjectIfNeededOnNode1 returns a byte slice containing the project name.
// rentBoskosProjectIfNeededOnNode1 also initializes boskosClient if necessary.
// When the tests run in parallel mode in Ginkgo, this rentBoskosProjectIfNeededOnNode1 runs only on
// Ginkgo node 1. The output should be shared with all other Gingko nodes so that they all use the same
// GCP project.
func rentBoskosProjectIfNeededOnNode1() []byte {
	if *project != "" {
		return []byte{}
	}

	fmt.Printf("Renting project from Boskos\n")
	boskosClient = client.NewClient(*jobName, *boskosServerURL)
	boskosRenewingTomb = tomb.NewTomb()

	ctx, cancel := context.WithTimeout(context.Background(), *boskosWaitDuration)
	defer cancel()
	p, err := boskosClient.AcquireWait(ctx, *boskosProjectType, "free", "busy")
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Unable to rent project from Boskos: %v\n", err))
	fmt.Printf("Rented project %q from Boskos\n", p.Name)

	go renewBoskosProject(boskosClient, p.Name, boskosRenewingTomb)

	return []byte(p.Name)
}

// acceptBoskosProjectIfNeededFromNode1 accepts a GCP project rented from Boskos by Ginkgo node 1.
//
// acceptBoskosProjectIfNeededFromNode1 takes the output of rentBoskosProjectIfNeededOnNode1.
// When the tests run in parallel mode in Ginkgo, this function runs on all Ginkgo nodes.
func acceptBoskosProjectIfNeededFromNode1(data []byte) {
	if *project != "" {
		return
	}

	boskosProject := string(data)
	fmt.Printf("Received Boskos project %q from Ginkgo node 1.\n", boskosProject)
	*project = boskosProject
}

func renewBoskosProject(boskosClient *client.Client, projectName string, boskosRenewingTomb *tomb.Tomb) {
	defer boskosRenewingTomb.Done()
	for {
		select {
		case <-time.Tick(5 * time.Minute):
			fmt.Printf("Renewing boskosProject %q\n", projectName)
			if err := boskosClient.UpdateOne(projectName, "busy", nil); err != nil {
				fmt.Printf("Failed to update status for project %q with Boskos: %v\n", projectName, err)
			}
		case <-boskosRenewingTomb.Stopping():
			return
		}
	}
}

// releaseBoskosResourcesOnNode1 releases all rented Boskos resources if there is any.
func releaseBoskosResourcesOnNode1() {
	if boskosClient == nil {
		return
	}
	boskosRenewingTomb.Stop()
	if !boskosClient.HasResource() {
		return
	}
	fmt.Printf("Releasing all Boskos resources.\n")
	err := boskosClient.ReleaseAll("dirty")
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to release project to Boskos: %v", err))
}

func TestMain(m *testing.M) {
	RegisterFailHandler(ginkgo.Fail)
	flag.Parse()

	os.Exit(m.Run())
}
