# Keycloakerator

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/statisticsnorway/keycloakerator/blob/main/LICENSE.md)

## Usage

To build and deploy locally you can follow the official Kubebuilder guide: [Running and deploying the controller](https://book.kubebuilder.io/cronjob-tutorial/running).
When deploying to a real cluster use the Kustomization located in `config/default`. You should set the required environment variables before deploying. The `controller` image to the correct one through either a patch or `kustomize edit set image controller=<image-ref>`.

### Environment variables

Set `KEYCLOAKERATOR_DUMMY=yes` in order to run the controller with a Keycloak dummy instead of a real instance. If this environment variable is set, the rest are not used.

| Name | Description | Required | Default |
|------|-------------|----------|---------|
| `KEYCLOAKERATOR_CLIENT_ID` | The Client ID used to authenticate with Keycloak. | x | N/A |
| `KEYCLOAKERATOR_CLIENT_SECRET` | The Client Secret used to authenticate with Keycloak. | x | N/A |
| `KEYCLOAKERATOR_CLIENT_REALM` | The realm the operator's Keycloak client belongs to. | | `master` |
| `KEYCLOAKERATOR_KEYCLOAK_URL` | The base url for Keycloak API calls. E.g. `http://keycloak.sso.svc.cluster.local` | x | N/A |

Kubebuilder also bundles a `ENABLE_WEBHOOKS` environment variable. Set this to `false` to disable webhooks (if testing/troubleshooting).
