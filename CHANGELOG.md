# Changelog

## [0.2.4](https://github.com/statisticsnorway/keycloakerator/compare/v0.2.3...v0.2.4) (2026-03-09)


### Bug Fixes

* changed kube-rbac-proxy image location ([#31](https://github.com/statisticsnorway/keycloakerator/issues/31)) ([652e81c](https://github.com/statisticsnorway/keycloakerator/commit/652e81c28af7a6e9ab85d7973797274ddc4a7770))
* **test:** remove outdated test case ([84392d4](https://github.com/statisticsnorway/keycloakerator/commit/84392d40ba02e9b18feadfb94fb6037537e0ad75))

## [0.2.3](https://github.com/statisticsnorway/keycloakerator/compare/v0.2.2...v0.2.3) (2024-12-09)


### Bug Fixes

* bump memory because OOMKilled ([#17](https://github.com/statisticsnorway/keycloakerator/issues/17)) ([aadf617](https://github.com/statisticsnorway/keycloakerator/commit/aadf617cca5b8b9ff6df4aa716f9668d4d694e05))

## [0.2.2](https://github.com/statisticsnorway/keycloakerator/compare/v0.2.1...v0.2.2) (2024-11-29)


### Bug Fixes

* adjusting memory keycloakerator manager ([#14](https://github.com/statisticsnorway/keycloakerator/issues/14)) ([ee9f370](https://github.com/statisticsnorway/keycloakerator/commit/ee9f370d4f1db5e37d957bef0ddb5868b08584c1))

## [0.2.1](https://github.com/statisticsnorway/keycloakerator/compare/v0.2.0...v0.2.1) (2024-06-10)


### Bug Fixes

* create protocol mapper for client audience ([#9](https://github.com/statisticsnorway/keycloakerator/issues/9)) ([acdeffd](https://github.com/statisticsnorway/keycloakerator/commit/acdeffd1fa4edde9e0ba9135492834aefa5939c4))

## [0.2.0](https://github.com/statisticsnorway/keycloakerator/compare/v0.1.0...v0.2.0) (2024-06-09)


### ⚠ BREAKING CHANGES

* use hyphens in secret keys\n\nBREAKING-CHANGES: secrets no longer use underscores in keys

### Features

* use hyphens in secret keys\n\nBREAKING-CHANGES: secrets no longer use underscores in keys ([848a122](https://github.com/statisticsnorway/keycloakerator/commit/848a122b4eb964e44905122d919a6bbf955b309f))

## [0.1.0](https://github.com/statisticsnorway/keycloakerator/compare/v0.0.1...v0.1.0) (2024-05-31)


### ⚠ BREAKING CHANGES

* rename and remove properties from spec

### Features

* add option to get client credentials from GCP SM ([b8ffd8f](https://github.com/statisticsnorway/keycloakerator/commit/b8ffd8f789d71979b6b459677e8b349b5edad356))


### Bug Fixes

* add default value for Keycloak in Reconciler ([2f709ea](https://github.com/statisticsnorway/keycloakerator/commit/2f709ea8413720461a6e9f1a0051e6fec5e55f5a))
* parse general config with prefix ([5e16b11](https://github.com/statisticsnorway/keycloakerator/commit/5e16b11d4a57b0e968ed6e72ee77bc0311b00b44))
* remove images from kustomization ([95a676f](https://github.com/statisticsnorway/keycloakerator/commit/95a676fbf1958b913c139753f8c7e0c81308ff39))
* use snake case for secret keys, use constant for key ref ([3155949](https://github.com/statisticsnorway/keycloakerator/commit/31559495fb4dfc813ec6d621e714542381240903))


### Code Refactoring

* rename and remove properties from spec ([9ef2231](https://github.com/statisticsnorway/keycloakerator/commit/9ef2231e26e9c66248941120a963feaaf91243bb))
