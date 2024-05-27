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

package controller

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/Nerzal/gocloak/v13"
	daplav1alpha1 "github.com/statisticsnorway/keycloakerator/api/v1alpha1"
)

// SimpleProxyClientReconciler reconciles a SimpleProxyClient object
type SimpleProxyClientReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Keycloak Keycloak
}

type SimpleProxyClientOption func(*SimpleProxyClientReconciler)

func NewSimpleProxyClientReconciler(mgr manager.Manager, opts ...SimpleProxyClientOption) *SimpleProxyClientReconciler {
	spc := &SimpleProxyClientReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}

	for _, opt := range opts {
		opt(spc)
	}

	return spc
}

func WithKeycloakDummy() SimpleProxyClientOption {
	return WithKeycloak(&KeycloakDummy{})
}

func WithKeycloak(kc Keycloak) SimpleProxyClientOption {
	return func(spcr *SimpleProxyClientReconciler) {
		spcr.Keycloak = kc
	}
}

type Keycloak interface {
	CreateClient(ctx context.Context, realm string, newClient gocloak.Client) (string, error)
	GetClientByClientId(ctx context.Context, realm string, clientId string) (*gocloak.Client, error)
	GetClient(ctx context.Context, realm, idOfClient string) (*gocloak.Client, error)
	DeleteClient(ctx context.Context, realm, idOfClient string) error
}

//+kubebuilder:rbac:groups=dapla.ssb.no,resources=simpleproxyclients,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dapla.ssb.no,resources=simpleproxyclients/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dapla.ssb.no,resources=simpleproxyclients/finalizers,verbs=update

//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

var finalizerName = fmt.Sprintf("%s/%s", daplav1alpha1.GroupVersion.Group, "simpleproxyclient")

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SimpleProxyClient object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *SimpleProxyClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.V(2).Info("starting reconciliaton")
	// Retrieve the SimpleProxyClient instance being reconciled
	instance := &daplav1alpha1.SimpleProxyClient{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.Info("could not get instance, possibly being deleted", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If instance is not being deleted, but does not have a finalizer set,
	// add the finalizer.
	if instance.GetDeletionTimestamp().IsZero() && !hasFinalizer(instance) {
		log.V(1).Info("adding finalizer")
		if err := r.addFinalizer(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	const clientIdFormat = "%s-%s"
	clientId := fmt.Sprintf(clientIdFormat, req.Namespace, req.Name)
	log = log.WithValues("clientId", clientId)

	// If the instance is being deleted, we need to delete the Keycloak client first.
	// The Secret is automatically garbage collected through its ownerReference.
	if !instance.GetDeletionTimestamp().IsZero() {
		log.V(2).Info("resource is being deleted")
		if hasFinalizer(instance) {
			log.V(2).Info("resource has finalizer, cleaning up resources and removing finalizer")
			// Delete Keycloak client
			client, err := r.Keycloak.GetClientByClientId(ctx, instance.Spec.Realm, clientId)

			// If we can't find the client, we don't need to delete it
			if err != nil && !errorIs[*ClientNotFoundError](err) {
				return ctrl.Result{}, err
			} else if err == nil {
				log.Info("deleting keycloak client")
				if err := r.Keycloak.DeleteClient(ctx, instance.Spec.Realm, *client.ID); err != nil {
					log.Error(err, "could not delete keycloak client")
					return ctrl.Result{}, err
				}
				log.Info("keycloak client deleted")
			}

			// We can now remove the finalizer, letting k8s remove the SimpleProxyClient instance
			log.V(1).Info("removing finalizer")
			if err := r.removeFinalizer(ctx, instance); err != nil {
				log.Error(err, "failed to remove finalizer")
				return ctrl.Result{}, err
			}
			log.Info("finalizer removed")

		}

		// Instance is being deleted, we don't need to run the rest of the reconcile logic.
		return ctrl.Result{}, nil
	}

	client, err := r.Keycloak.GetClientByClientId(ctx, instance.Spec.Realm, clientId)

	if errorIs[*ClientNotFoundError](err) {
		log.Info("creating keycloak client")
		clientInternalId, err := r.createClient(ctx, instance.Spec.Realm, clientId, instance.Spec.RedirectUris)
		if err != nil {
			return ctrl.Result{}, err
		}

		client, err = r.Keycloak.GetClient(ctx, instance.Spec.Realm, *clientInternalId)
		if err != nil {
			log.Error(err, "failed to get newly created client")
			return ctrl.Result{}, err
		}

	} else if err != nil {
		log.Error(err, "unexpected error getting client info (auth/connectivity issues with keycloak?)")
		return ctrl.Result{}, fmt.Errorf("get client: %w", err)
	} else if client.ID == nil {
		log.Error(err, "successfully retrieved client, but its client ID is missing")
		return ctrl.Result{}, fmt.Errorf("client internal ID for %q is nil (this should be impossible)", clientId)
	}

	if client == nil {
		log.Error(errors.New("client is nil"), "client was successfully retrieved or created, but client is nil (impossible?)")
		return ctrl.Result{}, fmt.Errorf("client for %q is nil (this should be impossible)", clientId)
	}

	if client.Secret == nil {
		log.Error(errors.New("client missing secret value"), "client exists, but is missing a client secret")
		return ctrl.Result{}, errors.New("client secret is nil")
	}
	clientSecret := client.Secret

	foundSecret := &corev1.Secret{}
	k8sSecret := &corev1.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Name:        instance.Spec.TargetSecret,
			Namespace:   instance.Namespace,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
		Data: map[string][]byte{
			"clientId":     []byte(clientId),
			"clientSecret": []byte(*clientSecret),
		},
	}
	secretLog := log.WithValues("secretName", instance.Spec.TargetSecret)
	if err = controllerutil.SetControllerReference(instance, k8sSecret, r.Scheme); err != nil {
		secretLog.Error(err, "failed to set controller reference on secret")
		return ctrl.Result{}, err
	}
	if err = r.Get(ctx,
		types.NamespacedName{Name: instance.Spec.TargetSecret, Namespace: instance.Namespace},
		foundSecret); k8serr.IsNotFound(err) {
		secretLog.Info("secret doesn't exist, creating it")
		cookieSecret, err := generateCookieSecret()
		if err != nil {
			return ctrl.Result{}, err
		}
		k8sSecret.Data["cookieSecret"] = []byte(cookieSecret)
		if err = r.Create(ctx, k8sSecret); err != nil {
			secretLog.Error(err, "could not create secret")
			return ctrl.Result{}, fmt.Errorf("create secret: %w", err)
		}
		return ctrl.Result{}, nil
	} else if err != nil {
		secretLog.Error(err, "unexpected error getting secret")
		return ctrl.Result{}, fmt.Errorf("get secret: %w", err)
	}

	if !areClientCredentialsCorrect(foundSecret, k8sSecret) {
		secretLog.Info("secret data diverges from wanted, updating with correct values")
		if _, ok := k8sSecret.Data["cookieSecret"]; !ok {
			cookieSecret, err := generateCookieSecret()
			if err != nil {
				return ctrl.Result{}, err
			}
			k8sSecret.Data["cookieSecret"] = []byte(cookieSecret)
		}

		if err = r.Update(ctx, k8sSecret); err != nil {
			secretLog.Error(err, "could not update secret")
			return ctrl.Result{}, fmt.Errorf("update secret: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

// Disgusting workaround to not being able to take references of primitives.
// Needed for gocloak.Client literal.
func ptr[T any](t T) *T {
	return &t
}

// errorIs is a convenience function for checking if an error is a specific custom error type
//
// Instead of checking the usual way,
//
//	var customErr *CustomError
//	if errors.As(e, customErr) {}
//
// you can use
//
//	if errorIs[CustomError](err) {}
func errorIs[T error](e error) bool {
	tErr := new(T)
	return errors.As(e, tErr)
}

// areClientCredentialsCorrect reports whether the client ID and secret are correct
func areClientCredentialsCorrect(found, want *corev1.Secret) bool {
	if clientId, ok := found.Data["clientId"]; !ok || !bytes.Equal(clientId, want.Data["clientId"]) {
		return false
	}
	if clientSecret, ok := found.Data["clientSecret"]; !ok || !bytes.Equal(clientSecret, want.Data["clientSecret"]) {
		return false
	}
	return true
}

func (r *SimpleProxyClientReconciler) createClient(ctx context.Context, realm, clientId string, redirectUris []string) (*string, error) {
	client := newClient(clientId, redirectUris)

	clientInternalId, err := r.Keycloak.CreateClient(ctx, realm, client)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	return &clientInternalId, nil
}

// newClient is a convenience function to create a gocloak.Client instance
// with our desired defaults.
func newClient(clientId string, redirectUris []string) gocloak.Client {
	return gocloak.Client{
		ClientID:     &clientId,
		Name:         &clientId,
		Enabled:      ptr(true),
		PublicClient: ptr(false),
		RedirectURIs: &redirectUris,
	}
}

func hasFinalizer(o client.Object) bool {
	return controllerutil.ContainsFinalizer(o, finalizerName)
}

func (r *SimpleProxyClientReconciler) addFinalizer(ctx context.Context, o client.Object) error {
	if ok := controllerutil.AddFinalizer(o, finalizerName); !ok {
		// This shouldn't really happen (we only call this function if the finalizer isn't present)
		return nil
	}
	return r.Update(ctx, o)
}

func (r *SimpleProxyClientReconciler) removeFinalizer(ctx context.Context, instance *daplav1alpha1.SimpleProxyClient) error {
	if ok := controllerutil.RemoveFinalizer(instance, finalizerName); !ok {
		// This shouldn't really happen (we only call this function if the finalizer is present)
		return nil
	}
	return r.Update(ctx, instance)
}

func generateCookieSecret() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SimpleProxyClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&daplav1alpha1.SimpleProxyClient{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
