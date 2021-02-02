# Release Process

These are notes to help follow a consistent release process. See something
important missing? Please submit a pull request to add anything else that would
be useful!

## Preparing for a release

There are a few steps that should be taken prior to creating the actual release
itself.

1. Collect changes since last release. This can be done by looking directly at
   merged commit messages (``git log [last_release_tag]...HEAD``), or by
   viewing the changes on GitHub ([example:
   https://github.com/kubernetes/node-problem-detector/compare/v0.8.6...master](https://github.com/kubernetes/node-problem-detector/compare/v0.8.6...master)).

1. Based on the changes to be included in the release, determine what the next
   release number should be. We strive to follow [SemVer](https://semver.org/)
   as much as possible.

1. Update [CHANGELOG](https://github.com/kubernetes/node-problem-detector/blob/master/CHANGELOG.md)
   with all significant changes.

## Create release

Once changes have been merged to the CHANGELOG, perform the actual release via
GitHub. When creating the release, make sure to include the following in the
body of the release:

1. For convenience, add a link to easily view the changes since the last
   release (e.g.
   [https://github.com/kubernetes/node-problem-detector/compare/v0.8.5...v0.8.6](https://github.com/kubernetes/node-problem-detector/compare/v0.8.5...v0.8.6)).

1. There is no need to duplicate everything from the CHANGELOG, but include the
   most significant things so someone just viewing the release entry will have
   an idea of what it includes.

1. Provide a link to the new image release (e.g. `Image:
   k8s.gcr.io/node-problem-detector/node-problem-detector:v0.8.6`)

## Post release steps

1. Update image version in
   [deployment/node-problem-detector.yaml](https://github.com/kubernetes/node-problem-detector/blob/422c088d623488be33aa697588655440c4e6a063/deployment/node-problem-detector.yaml#L32).

   Update the image version in the deployment file so anyone deploying directly
   from the repo deployment file will get the newest image deployed.
