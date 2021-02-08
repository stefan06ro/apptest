# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).



## [Unreleased]

### Fixed

- Extend support for setting the app CR name.

## [0.10.1] - 2021-02-08

### Fixed

- Revert `sigs.k8s.io/controller-runtime` to v0.6.4.

## [0.10.0] - 2021-02-03

### Added

- Add support for setting the app CR name.
- Set `app.kubernetes.io/name` label for app CRs.

## [0.9.0] - 2020-12-15

### Added

- Adding App Upgrade test.

## [0.8.2] - 2020-12-14

### Fixed

- Fix namespace handling when waiting for deployed apps.

## [0.8.1] - 2020-12-14

### Fixed

- Fix namespace handling when waiting for deployed apps.

## [0.8.0] - 2020-12-10

### Added

- Add support for setting kubeconfig secret in app CRs for remote clusters.
- Add clean up function to remove resources created while installing apps.

### Changed

- User config values are created on App namespace.

## [0.7.1] - 2020-11-26

### Fixed

- Comparing `SHA` parameter with either app version or version.

## [0.7.0] - 2020-11-17

### Fixed

- Install specified app version instead of latest when passing the `SHA` parameter.

### Changed

- Updated `appcatalog` library.

## [0.6.0] - 2020-11-12

### Changed

- Don't fail when ensuring a CRD that's already present.

## [0.5.0] - 2020-11-06

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

[Unreleased]: https://github.com/giantswarm/apptest/compare/v0.10.1...HEAD
[0.10.1]: https://github.com/giantswarm/apptest/compare/v0.10.0...v0.10.1
[0.10.0]: https://github.com/giantswarm/apptest/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/giantswarm/apptest/compare/v0.8.2...v0.9.0
[0.8.2]: https://github.com/giantswarm/apptest/compare/v0.8.1...v0.8.2
[0.8.1]: https://github.com/giantswarm/apptest/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/giantswarm/apptest/compare/v0.7.1...v0.8.0
[0.7.1]: https://github.com/giantswarm/apptest/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/giantswarm/apptest/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/giantswarm/apptest/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/giantswarm/apptest/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/giantswarm/apptest/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/giantswarm/apptest/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/giantswarm/apptest/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/giantswarm/apptest/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/giantswarm/apptest/releases/tag/v0.1.0
