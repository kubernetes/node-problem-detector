# Versioning Scheme for Node Problem Detector

This document describes the versioning scheme for the Node Problem Detector, which is designed to align with Kubernetes releases.

## Versioning Scheme: `v1.<k8s_minor>.<patch>`

The versioning scheme is `v1.<k8s_minor>.<patch>`.

* **v1**: The major version is 1, indicating the project is considered stable.
* **`<k8s_minor>`**: This part of the version is the minor version of the supported Kubernetes release. For example, for Kubernetes v1.34, this would be `34`.
* **`<patch>`**: This is a patch number for bug fixes and other small changes within the same supported Kubernetes version.

For example, for Kubernetes v1.34, the corresponding version of the Node Problem Detector is in the `v1.34.x` series. The first release for this Kubernetes version would be `v1.34.0`.

The release branch is named `release-1.<k8s_minor>` to match Kubernetes versions (e.g. `release-1.34`). Git tags are named after the version, like `v1.<k8s_minor>.<patch>`. Patches are created from these release branches.

This scheme is not ambiguous and will work for the foreseeable future. Kubernetes v1.100 is not expected until approximately 2047.

## Reasoning

This versioning scheme provides the following benefits:

* **Clarity**: It is immediately clear which version of Node Problem Detector is compatible with which version of Kubernetes.
* **Consistency**: It aligns the project with the release cycle of Kubernetes and some other Kubernetes components versioning.
* **Predictability**: Users can better predict when new releases will be available.
* **Easier Maintenance**: By having separate version lines for each Kubernetes minor version (e.g., `v1.34.x`, `v1.35.x`), we can easily backport critical bug fixes and CVEs to older, still-supported release lines without being forced to also backport newer features.
* **Targeted Testing**: Each version of Node Problem Detector is tested against a specific Kubernetes version. This also implies testing against particular versions of related components like the container runtime and OS. New features in Node Problem Detector will not necessarily be tested against older versions of these components.

## Previous Versioning Scheme

For reference, the previous versioning scheme was `vX.Y.Z`.

* The major version `X` was always `0`.
* The minor version `Y` was incremented for releases with significant new features.
* The patch version `Z` was used for smaller features and bug fixes.

This model did not provide a clear link to the supported Kubernetes version, making it difficult for users to determine compatibility. We will have a branch named `release-0.8` so we can keep track of changes that are used as hotfixes for `0.8.21` and below releases.
