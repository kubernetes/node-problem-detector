# Security Policy

## Security Announcements

Join the [kubernetes-security-announce] group for security and vulnerability announcements.

You can also subscribe to an RSS feed of the above using [this link][kubernetes-security-announce-rss].

## Reporting a Vulnerability

Instructions for reporting a vulnerability can be found on the [Kubernetes Security and Disclosure Information] page.

## Dependency CVEs and vulnerability scanner findings

node-problem-detector is a Go program with a large dependency tree, and vulnerability scanners regularly flag CVEs in its dependencies. The project follows the Kubernetes-wide guidance for handling these reports — see [CVEs in our dependencies] in the Kubernetes security guide. In short:

- Scanners match dependency versions; they do not check whether the vulnerable code is reachable. The large majority of CVEs flagged against node-problem-detector dependencies are in code paths its binaries never execute. Reachability evidence (for example, `govulncheck` output showing a call stack to the vulnerable symbol) is what determines whether a finding is treated as a vulnerability in node-problem-detector.
- Dependency updates land on `master` continuously and ship with the next scheduled release (see [versioning](docs/versioning.md) for the release cadence). The project is maintained by a small group of volunteers and does not cut out-of-band releases solely to update dependency version strings for findings that are not reachable.
- If you have evidence that a CVE is reachable and exploitable in node-problem-detector as deployed, please report it through the process above. That is treated as a vulnerability in node-problem-detector itself, and a fix will be prioritized and backported to supported release branches.

If your compliance program requires container images with zero scanner findings on a faster cadence than the project's releases, rebuild node-problem-detector from the latest release tag or from `master` in your own build pipeline — everything needed is in this repository (see the [Makefile](Makefile) and [Dockerfile](Dockerfile)). No upstream release cadence can track every scanner database, and rebuilding downstream is standard practice for organizations with strict scanning requirements.

## Supported Versions

Fixes land on `master` first. Release lines follow supported Kubernetes minor versions as described in [versioning](docs/versioning.md), and fixes for vulnerabilities that are reachable in node-problem-detector may be backported to supported release branches.

[kubernetes-security-announce]: https://groups.google.com/forum/#!forum/kubernetes-security-announce
[kubernetes-security-announce-rss]: https://groups.google.com/forum/feed/kubernetes-security-announce/msgs/rss_v2_0.xml?num=50
[Kubernetes Security and Disclosure Information]: https://kubernetes.io/docs/reference/issues-security/security/#report-a-vulnerability
[CVEs in our dependencies]: https://github.com/kubernetes/community/blob/master/contributors/guide/security.md#cves-in-our-dependencies
