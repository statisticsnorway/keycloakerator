# Keycloakerator

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/statisticsnorway/keycloakerator/blob/main/LICENSE.md)

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
