# Release Process

These are notes to help follow a consistent release process. See something
important missing? Please submit a pull request to add anything else that would
be useful!

## Prerequisites

Ensure access to the container image [staging registry](https://console.cloud.google.com/gcr/images/k8s-staging-npd/global/node-problem-detector).
Add email to `k8s-infra-staging-npd` group in sig-node [groups.yaml](https://github.com/kubernetes/k8s.io/blob/main/groups/sig-node/groups.yaml).
See example https://github.com/kubernetes/k8s.io/pull/1599.

## Preparing for a release

There are a few steps that should be taken prior to creating the actual release
itself.

1. Collect changes since last release. This can be done by looking directly at
   merged commit messages (``git log [last_release_tag]...HEAD``), or by
   viewing the changes on GitHub (example: https://github.com/kubernetes/node-problem-detector/compare/v0.8.15...master).

2. Based on the changes to be included in the release, determine what the next
   release number should be. We strive to follow [SemVer](https://semver.org/)
   as much as possible.

3. Update [CHANGELOG](https://github.com/kubernetes/node-problem-detector/blob/master/CHANGELOG.md)
   with all significant changes.

## Create release

### Create the new version tag

#### Option 1
```
# Use v0.8.17 as an example.
git clone git@github.com:kubernetes/node-problem-detector.git
cd node-problem-detector/
git tag v0.8.17
git push origin v0.8.17
```

#### Option 2
Update [version.txt](https://github.com/kubernetes/node-problem-detector/blob/master/version.txt)
(example https://github.com/kubernetes/node-problem-detector/pull/869).

### Build and push artifacts
This step builds the NPD into container files and tar files.
- The container file is pushed to the [staging registry](https://console.cloud.google.com/gcr/images/k8s-staging-npd/global/node-problem-detector).
  You will promote the new image to registry.k8s.io later.
- The tar files are generated locally. You will upload those to github in the
  release note later.

**Note: You need the access mentioned in the [prerequisites](#prerequisites)
section to perform steps in this section.**

```
# One-time setup
sudo  apt-get install libsystemd-dev gcc-aarch64-linux-gnu

cd node-problem-detector
make release

# Get SHA256 of the tar files. For example
sha256sum node-problem-detector-v0.8.17-linux_amd64.tar.gz
sha256sum node-problem-detector-v0.8.17-linux_arm64.tar.gz
sha256sum node-problem-detector-v0.8.17-windows_amd64.tar.gz

# Get MD5 of the tar files. For example
md5sum node-problem-detector-v0.8.17-linux_amd64.tar.gz
md5sum node-problem-detector-v0.8.17-linux_arm64.tar.gz
md5sum node-problem-detector-v0.8.17-windows_amd64.tar.gz

# Verify container image in staging registry and get SHA256.
docker pull gcr.io/k8s-staging-npd/node-problem-detector:v0.8.17
docker image ls gcr.io/k8s-staging-npd/node-problem-detector --digests
```

### Promote new NPD image to registry.k8s.io
1. Get the SHA256 from the new NPD image from the [staging registry](https://console.cloud.google.com/gcr/images/k8s-staging-npd/global/node-problem-detector)
   or previous step.
2. Promote the NPD image to registry.k8s.io ([images.yaml](https://github.com/kubernetes/k8s.io/blob/main/registry.k8s.io/images/k8s-staging-npd/images.yaml), example https://github.com/kubernetes/k8s.io/pull/6523).
3. Verify the container image.
```
docker pull registry.k8s.io/node-problem-detector/node-problem-detector:v0.8.17
docker image ls registry.k8s.io/node-problem-detector/node-problem-detector:v0.8.17
```

### Create the release note

Go to https://github.com/kubernetes/node-problem-detector/releases, draft a new
release note and publish. Make sure to include the following in the body of the
release note:

1. For convenience, add a link to easily view the changes since the last
   release (e.g.
   [https://github.com/kubernetes/node-problem-detector/compare/v0.8.15...v0.8.17](https://github.com/kubernetes/node-problem-detector/compare/v0.8.15...v0.8.17)).

2. There is no need to duplicate everything from the CHANGELOG, but include the
   most significant things so someone just viewing the release entry will have
   an idea of what it includes.

3. Provide a link to the new image release (e.g. `Image:
   registry.k8s.io/node-problem-detector/node-problem-detector:v0.8.17`)

4. Upload the tar files built from [pevious step](#build-and-push-artifacts),
   and include the SHA and MD5.

## Post release steps

1. Update image version in node-problem-detector repo, so anyone deploying
   directly from the repo deployment file will get the newest image deployed.
   Example https://github.com/kubernetes/node-problem-detector/pull/897.

2. Update the NPD version in [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes)
   repo, so that kubernetes clusters use the new NPD version. Example
   https://github.com/kubernetes/kubernetes/pull/123740.
