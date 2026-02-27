# Keycloakerator

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/statisticsnorway/keycloakerator/blob/main/LICENSE.md)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://conventionalcommits.org)

## What it does

Keycloakerator is a small Kubernetes controller (operator) that automates the lifecycle
of Keycloak OpenID Connect clients for services that use a simple auth-proxy pattern.
It exposes a Custom Resource Definition (CRD) named `SimpleProxyClient` which lets
you declare the desired redirect URIs and the name of a Kubernetes `Secret` that
will hold the resulting client credentials.

When a `SimpleProxyClient` is created, the controller will:

- ensure a corresponding client exists in Keycloak (creating it if missing),
- create or update a Kubernetes `Secret` containing the `client_id`, `client_secret`
  and a `cookie` secret used by the proxy,
- set controller ownership so the secret is managed alongside the CR, and
- cleanup the Keycloak client on CR deletion (via a finalizer).

The Keycloak interaction is implemented via a small wrapper around `gocloak` in
`internal/keycloak`. The controller itself lives in `internal/controller` and the
API types and webhook validation are defined under `api/v1alpha1`.

## Background / Why this exists

Manually creating and configuring Keycloak clients for each service is error
prone and does not fit well with a GitOps / declarative workflow. Keycloakerator
lets teams declare their client requirements alongside their Kubernetes
applications so client creation, credentials rotation and cleanup happen
automatically and reproducibly.

Keycloakerator was created to simplify the common pattern of services sitting
behind a simple authentication proxy that needs an OIDC client. By managing the
client and the resulting secret through Kubernetes custom resources, teams get
consistent client configuration and a clear lifecycle for credentials.

Important operational notes:

- The controller requires a Keycloak client with permissions to manage clients
  on the target realm (see configuration section below).
- For testing, a dummy Keycloak implementation can be enabled via
  the `KEYCLOAK_DUMMY` flag.
- Example usage: install the CRD from `config/crd/bases` and apply a sample from
  `config/samples/dapla_v1alpha1_simpleproxyclient.yaml`.

## Usage

To build and deploy locally you can follow the official Kubebuilder guide: [Running and deploying the controller](https://book.kubebuilder.io/cronjob-tutorial/running).
When deploying to a real cluster use the Kustomization located in `config/default`. You should set the required environment variables before deploying. The `controller` image to the correct one through either a patch or `kustomize edit set image controller=<image-ref>`.

### Configuration

Keycloakerator needs a Keycloak client with Service Account roles enabled, and
granted `manage-clients` on the realm it is going to manage. You can provide
the necessary parameters in a couple of ways.

1. Through environment variables
  a. Set `KEYCLOAKERATOR_CLIENT_ID` to the client ID (friendly ID, not the UUID)
  b. Set `KEYCLOAKERATOR_CLIENT_SECRET` to the client secret
  c. Set `KEYCLOAKERATOR_CLIENT_REALM` to the client's realm
2. Through GCP Secret Manager
  a. Set `KEYCLOAKERATOR_GCP_SECRET` to the full resource name of a GCP Secret
      Manager secret containing a YAML file with the fields `client_id`,
      `client_secret` and `client_realm`. All fields are optional.

You can also mix these: the Secret Manager values are read first, and then
overridden by any environment variables that are set.

You must also set `KEYCLOAKERATOR_KEYCLOAK_URL` to the base URL of the
Keycloak instance, e.g. `https://keycloak.example.com`.

Set `KEYCLOAKERATOR_DUMMY=true` in order to run the controller with a Keycloak dummy instead of a real instance.

Kubebuilder also bundles a `ENABLE_WEBHOOKS` environment variable. Set this to `false` to disable webhooks (if testing/troubleshooting).

## Contributing

Please follow these guidelines when contributing.

## Commit messages and merging PRs

Use squash merges, not merge commits.
This allows the release-please workflow to parse them and create a changelog.

This project follows [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for its commit messages - **this also applies to squash merge messages**.
You can check out the following resources for more explanation/motivation:
[The power of conventional commits](https://julien.ponge.org/blog/the-power-of-conventional-commits/)
 and
[Conventional Commit Messages](https://gist.github.com/qoomon/5dfcdf8eec66a051ecd85625518cfd13).

When working on experimental branches you can use whatever commit messages you want, but you should either squash/amend your messages before merging your PR.
Using [Scratchpad branches](https://julien.ponge.org/blog/a-workflow-for-experiments-in-git-scratchpad-branches/) is probably the easiest approach.

Use the provided pre-commit hook to verify your commit messages:

```sh
pre-commit install --install-hooks
pre-commit install -t commit-msg
```

## Creating a release

Google's [release-please](https://github.com/googleapis/release-please) is used to create releases.
release-please maintains a release PR, which determines the next semver version based on whether there have been feature additions, breaking changes, etc.
To create a release, simply merge that PR, and it will create a GitHub release, tag and a Docker image will be built.

The suggested next version can be overriden by including `Release-As: x.x.x` in a commit message. For example:

```sh
git commit --allow-empty -m "chore: release 2.0.0" -m "Release-As: 2.0.0"
```
