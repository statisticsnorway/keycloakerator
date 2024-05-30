/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/url"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"golang.org/x/oauth2/clientcredentials"
	"gopkg.in/yaml.v3"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/caarlos0/env/v11"
	daplav1alpha1 "github.com/statisticsnorway/keycloakerator/api/v1alpha1"
	"github.com/statisticsnorway/keycloakerator/internal/controller"
	//+kubebuilder:scaffold:imports
)

// Config for the overall controller
type config struct {
	// Whether to use the dummy implementation of the Keycloak interface
	KeycloakDummy bool `env:"KEYCLOAK_DUMMY"`

	// If set, will try to populate keycloakConfig using this secret
	GCPSecret string `env:"GCP_SECRET"`
}

// Config parameters for Keycloak
type keycloakConfig struct {
	// Keycloak Client ID
	ClientId string `env:"CLIENT_ID" yaml:"client_id"`

	// Keycloak Client Secret
	ClientSecret string `env:"CLIENT_SECRET" yaml:"client_secret"`

	// Realm of the above Keycloak client
	ClientRealm string `env:"CLIENT_REALM" yaml:"client_realm"`

	// Base URL for the Keycloak instance
	KeycloakUrl url.URL `env:"KEYCLOAK_URL,required,notEmpty" yaml:"-"`
}

func (kc *keycloakConfig) Valid() bool {
	return kc.ClientId != "" && kc.ClientSecret != "" && kc.ClientRealm != "" && kc.KeycloakUrl.String() != ""
}

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(daplav1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "8419679a.ssb.no",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	ctx := ctrl.SetupSignalHandler()

	cfg, err := env.ParseAsWithOptions[config](env.Options{Prefix: "KEYCLOAKERATOR_"})
	if err != nil {
		setupLog.Error(err, "unable to parse general config from env")
	}

	var ctrlOpts []controller.SimpleProxyClientOption

	if !cfg.KeycloakDummy {
		kcConfig := &keycloakConfig{}

		if cfg.GCPSecret != "" {
			if err = readKeycloakConfigFromSecretManager(ctx, cfg.GCPSecret, kcConfig); err != nil {
				setupLog.Error(err, "unable to fetch or parse config from secret manager")
			}
		}

		err = env.ParseWithOptions(kcConfig, env.Options{
			Prefix: "KEYCLOAKERATOR_",
		})
		if err != nil {
			fmt.Printf("error parsing environment variables: %s", err)
			os.Exit(1)
		}

		if !kcConfig.Valid() {
			fmt.Print("missing one or more keycloak parameters")
			os.Exit(1)
		}

		setupLog.Info("initializing GoCloak wrapper")

		// The oauth2/clientcredentials package provides a TokenSource which keeps our Keycloak token
		// up to date automatically.
		authConfig := &clientcredentials.Config{
			ClientID:     kcConfig.ClientId,
			ClientSecret: kcConfig.ClientSecret,
			TokenURL: kcConfig.KeycloakUrl.JoinPath(
				"realms",
				kcConfig.ClientRealm,
				"protocol/openid-connect/token",
			).String(),
		}

		kc := controller.NewGocloakWrapper(
			kcConfig.KeycloakUrl.String(),
			kcConfig.ClientRealm,
			authConfig.TokenSource(ctx),
		)
		ctrlOpts = append(ctrlOpts, controller.WithKeycloak(kc))
	}

	// Set up our reconciler
	if err = controller.NewSimpleProxyClientReconciler(mgr, ctrlOpts...).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SimpleProxyClient")
		os.Exit(1)
	}
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		setupLog.Info("creating webhook")
		if err = (&daplav1alpha1.SimpleProxyClient{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "SimpleProxyClient")
			os.Exit(1)
		}
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func readKeycloakConfigFromSecretManager(ctx context.Context, secret string, kc *keycloakConfig) error {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secret,
	}
	resp, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(resp.GetPayload().GetData(), kc)
}
