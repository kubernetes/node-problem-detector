# Node Problem Detector End-To-End tests

NPD e2e tests are meant for testing the NPD on a VM environment.

Currently the tests only support Google Compute Engine (GCE) environment. Support for other vendors can be added in future.

## Prerequisites

1. Setup [Google Application Default Credentials (ADC)](https://developers.google.com/identity/protocols/application-default-credentials), which is [required for authentication](https://godoc.org/google.golang.org/api/compute/v1#hdr-Creating_a_client) by the Compute Engine API.
2. Setup a [project-wide SSH key](https://cloud.google.com/compute/docs/instances/adding-removing-ssh-keys#project-wide) that can be used to SSH into the GCE VMs.

## Running tests

From the node-problem-detector base directory, run:

```
export GOOGLE_APPLICATION_CREDENTIALS=[YOUR_ADC_PATH:~/.config/gcloud/application_default_credentials.json]
export ZONE=[ANY_GCE_ZONE:us-central1-a]
export PROJECT=[YOUR_PROJECT_ID]
export VM_IMAGE=[TESTED_OS_IMAGE:cos-73-11647-217-0]
export IMAGE_PROJECT=[TESTED_OS_IMAGE_PROJECT:cos-cloud]
export SSH_USER=${USER}
export SSH_KEY=~/.ssh/id_rsa
export ARTIFACTS=/tmp/npd
make e2e-test
```
