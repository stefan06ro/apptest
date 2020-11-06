# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).



## [Unreleased]

### Fixed

- Add new methods to the interface.

### Added

- Expose method `EnsureCRDs` to register CRDs in the k8s API.
- A custom `Scheme` can be passed to configure the controller-runtime client.
- Add getter method that returns the controller-runtime client.
- Generate catalog URLs for known catalogs.

## [0.4.1] - 2020-10-30

### Added

- Support both explicit kubeconfigs and file paths.

### Fixed

- Optimize apps wait interval as app-operator has a status webhook.

## [0.4.0] - 2020-10-29

### Changed

- Remove k8sclient dependency and use controller-runtime client for managing CRs.

## [0.3.0] - 2020-10-08

### Added

- Add support for configuring app CRs with values.

## [0.2.0] - 2020-10-06

### Added

- Add support for setting app version from SHA for test catalogs.

## [0.1.0] - 2020-09-30

### Added

- Add initial version that implements InstallApps for use in apptestctl and
Go integration tests.

[Unreleased]: https://github.com/giantswarm/apptest/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/giantswarm/apptest/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/giantswarm/apptest/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/giantswarm/apptest/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/giantswarm/apptest/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/giantswarm/apptest/releases/tag/v0.1.0
