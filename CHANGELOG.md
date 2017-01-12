# Change Log
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]
### Added
- Add travis presubmit test.

### Changed
- Update kubernetes version to v1.4.0-beta.3

## [0.2.0] - 2016-08-23
### Added
- Add look back support in kernel monitor. Kernel monitor will look back for
  specified amount of time to detect old problems during each start or restart.
- Add support for some kernel oops detection.

### Changed
- Change NPD to get node name from `NODE_NAME` env first before `os.Hostname`,
  and update the example to get node name from downward api and set `NODE_NAME`.

## 0.1.0 - 2016-06-09
### Added
- Initial version of node problem detector.

[Unreleased]: https://github.com/kubernetes/node-problem-detector/compare/v0.2...HEAD
[0.2.0]: https://github.com/kubernetes/node-problem-detector/compare/v0.1...v0.2
