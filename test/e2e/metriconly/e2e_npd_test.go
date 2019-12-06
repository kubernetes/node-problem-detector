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

func TestNPD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var err error
	computeService, err = gce.GetComputeClient()
	if err != nil {
		panic(fmt.Sprintf("Unable to create gcloud compute service using defaults. Make sure you are authenticated. %v", err))
	}

	if *project == "" {
		boskosClient := client.NewClient(*jobName, *boskosServerURL)
		*project = acquireProjectOrDie(boskosClient)

		defer releaseProjectOrDie(boskosClient)
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

func acquireProjectOrDie(boskosClient *client.Client) string {
	fmt.Printf("Renting project from Boskos\n")
	ctx, cancel := context.WithTimeout(context.Background(), *boskosWaitDuration)
	defer cancel()
	p, err := boskosClient.AcquireWait(ctx, *boskosProjectType, "free", "busy")
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Unable to rent project from Boskos: %v\n", err))

	fmt.Printf("Rented project %s from Boskos", p.Name)

	go func(boskosClient *client.Client, projectName string) {
		for range time.Tick(5 * time.Minute) {
			if err := boskosClient.UpdateOne(projectName, "busy", nil); err != nil {
				fmt.Printf("Failed to update status for project %s with Boskos: %v\n", projectName, err)
			}
		}
	}(boskosClient, p.Name)

	return p.Name
}

func releaseProjectOrDie(boskosClient *client.Client) {
	if !boskosClient.HasResource() {
		return
	}
	err := boskosClient.ReleaseAll("dirty")
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to release project to Boskos: %v", err))
}

func TestMain(m *testing.M) {
	RegisterFailHandler(ginkgo.Fail)
	flag.Parse()

	os.Exit(m.Run())
}
