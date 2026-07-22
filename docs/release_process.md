# Release Process

These are notes to help follow a consistent release process. See something important missing? Please submit a pull request to add anything else that would be useful!

## Prerequisites

Ensure access to the container image [staging registry](https://console.cloud.google.com/gcr/images/k8s-staging-npd/global/node-problem-detector).
Add email to `k8s-infra-staging-npd` group in sig-node [groups.yaml](https://github.com/kubernetes/k8s.io/blob/main/groups/sig-node/groups.yaml).
See example https://github.com/kubernetes/k8s.io/pull/1599.

The steps below also require the following tools:

- [crane](https://github.com/google/go-containerregistry/blob/main/cmd/crane/README.md), to read image digests from the registries.
- Docker and a Go toolchain, to build the release tarballs with `make release-new`.

## Preparing for a release

There are a few steps that should be taken prior to creating the actual release itself.

1. Collect changes since last release. This can be done by looking directly at merged commit messages (``git log [last_release_tag]...HEAD``), or by viewing the changes on GitHub (example: https://github.com/kubernetes/node-problem-detector/compare/v1.35.0...master).

2. Based on the changes to be included in the release, determine what the next release number should be. We strive to follow [SemVer](https://semver.org/) as much as possible.

3. Update [CHANGELOG](https://github.com/kubernetes/node-problem-detector/blob/master/CHANGELOG.md) with all significant changes.

## Create release

### Create the new version tag

Update [version.txt](https://github.com/kubernetes/node-problem-detector/blob/master/version.txt) (example https://github.com/kubernetes/node-problem-detector/pull/1312).

### Container images are built automatically

Pushing the tag triggers a Cloud Build job ([cloudbuild.yaml](../cloudbuild.yaml)) that runs `make push-container` and `make push-container-windows`, which build and push both multi-arch images to the staging registry automatically:

- `gcr.io/k8s-staging-npd/node-problem-detector:v1.36.0` (linux/amd64, linux/arm64)
- `gcr.io/k8s-staging-npd/node-problem-detector-windows:v1.36.0` (windows/amd64)

No local container build or push (`make release`) is needed. Verify the images in the [staging registry](https://console.cloud.google.com/gcr/images/k8s-staging-npd/global/node-problem-detector) before continuing.

### Promote the NPD images to registry.k8s.io

1. Get the digests of the new NPD images from the staging registry:
```
crane digest gcr.io/k8s-staging-npd/node-problem-detector:v1.36.0
crane digest gcr.io/k8s-staging-npd/node-problem-detector-windows:v1.36.0
```
2. Promote both images to registry.k8s.io by adding the digests to [images.yaml](https://github.com/kubernetes/k8s.io/blob/main/registry.k8s.io/images/k8s-staging-npd/images.yaml) in the kubernetes/k8s.io repo: the linux digest under the `node-problem-detector` entry and the windows digest under the `node-problem-detector-windows` entry (example https://github.com/kubernetes/k8s.io/pull/9707).
3. After the promotion PR merges, verify that the promoted digests match staging:
```
crane digest registry.k8s.io/node-problem-detector/node-problem-detector:v1.36.0
crane digest registry.k8s.io/node-problem-detector/node-problem-detector-windows:v1.36.0
```

### Build the release artifacts

This step runs **after** the image promotion, because the release binaries are extracted from the promoted registry.k8s.io images.

```
# One-time setup on Linux hosts (journald build tags for the test binaries).
sudo apt-get install libsystemd-dev gcc-aarch64-linux-gnu

cd node-problem-detector
make release-new VERSION=v1.36.0
```

Only `VERSION` needs to be set (`TAG` and `NPD_NAME_VERSION` are derived from it), so there is no need to check out the release tag. `make release-new`:

- Pulls the promoted `registry.k8s.io/node-problem-detector/node-problem-detector:v1.36.0` and `.../node-problem-detector-windows:v1.36.0` images and extracts the release binaries from them (`docker create` + `docker cp`).
- Packages `node-problem-detector-v1.36.0-linux_amd64.tar.gz`, `node-problem-detector-v1.36.0-linux_arm64.tar.gz` and `node-problem-detector-v1.36.0-windows_amd64.tar.gz`, each with a `.sha512` file next to it.
- Prints the SHA256 and MD5 of the tarballs ([hack/print-tar-sha-md5.sh](../hack/print-tar-sha-md5.sh)), ready to paste into the release note.

### Create the release note

Go to https://github.com/kubernetes/node-problem-detector/releases, draft a new release note and publish. Make sure to include the following in the body of the release note:

1. For convenience, add a link to easily view the changes since the last release (e.g. [https://github.com/kubernetes/node-problem-detector/compare/v1.35.0...v1.36.0](https://github.com/kubernetes/node-problem-detector/compare/v1.35.0...v1.36.0)).

2. There is no need to duplicate everything from the CHANGELOG, but include the most significant things so someone just viewing the release entry will have an idea of what it includes.

3. Provide a link to the new image release (e.g. `Image: registry.k8s.io/node-problem-detector/node-problem-detector:v1.36.0`)

4. Upload the tar files built in the [previous step](#build-the-release-artifacts), and include the SHA and MD5 printed by `make release-new`.

## Post release steps

1. Update image version in node-problem-detector repo, so anyone deploying directly from the repo deployment file will get the newest image deployed. Example https://github.com/kubernetes/node-problem-detector/pull/897.

2. Update the NPD version in [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes) repo, so that kubernetes clusters use the new NPD version. Example https://github.com/kubernetes/kubernetes/pull/123740.
