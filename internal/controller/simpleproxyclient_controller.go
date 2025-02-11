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
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	daplav1alpha1 "github.com/statisticsnorway/keycloakerator/api/v1alpha1"
)

// SimpleProxyClientReconciler reconciles a SimpleProxyClient object
type SimpleProxyClientReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	oidcService OIDCService
}

const (
	clientSecretKey = "client-secret"
	clientIdKey     = "client-id"
	cookieSecretKey = "cookie-secret"

	daplaGroupAnnotation = "dapla.ssb.no/access-group"
)

func NewSimpleProxyClientReconciler(mgr manager.Manager, oidcService OIDCService) *SimpleProxyClientReconciler {
	spc := &SimpleProxyClientReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		oidcService: oidcService,
	}

	return spc
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

	clientId := toClientId(req.Name, req.Namespace)
	log = log.WithValues("clientId", clientId)

	// If the instance is being deleted, we need to delete the Keycloak client first.
	// The Secret is automatically garbage collected through its ownerReference.
	if !instance.GetDeletionTimestamp().IsZero() {
		log.V(2).Info("resource is being deleted")
		if !hasFinalizer(instance) {
			return ctrl.Result{}, nil
		}

		log.V(2).Info("resource has finalizer, cleaning up resources and removing finalizer")
		if err := r.deleteKeycloakClient(ctx, clientId); err != nil {
			log.Error(err, "failed to delete keycloak client")
			return ctrl.Result{}, err
		}

		// We can now remove the finalizer, letting k8s remove the SimpleProxyClient instance
		log.V(1).Info("removing finalizer")
		if err := r.removeFinalizer(ctx, instance); err != nil {
			log.Error(err, "failed to remove finalizer")
			return ctrl.Result{}, err
		}
		log.Info("finalizer removed")
		return ctrl.Result{}, nil
	}

	client, err := r.oidcService.GetClient(ctx, GetClientRequest{
		Name: clientId,
	})

	group, hasGroup := instance.Annotations[daplaGroupAnnotation]
	if !hasGroup {
		group = "UNKNOWN"
	}

	if errors.Is(err, ErrNotFound) {
		log.Info("creating keycloak client")
		if client, err = r.oidcService.CreateClient(ctx, CreateClientRequest{
			Name:         clientId,
			DaplaUser:    strings.TrimPrefix(req.Namespace, "user-ssb-"),
			DaplaGroup:   group,
			RedirectURIs: instance.Spec.RedirectUris,
		}); err != nil {
			log.Error(err, "could not create oidc client")
			return ctrl.Result{}, fmt.Errorf("create client: %w", err)
		}
	} else if err != nil {
		log.Error(err, "could not retrieve client info")
		return ctrl.Result{}, fmt.Errorf("get client: %w", err)
	}

	secret := &corev1.Secret{}
	secretLog := log.WithValues("secretName", instance.Spec.SecretName)
	if err = r.Get(ctx,
		types.NamespacedName{Name: instance.Spec.SecretName, Namespace: instance.Namespace},
		secret); k8serr.IsNotFound(err) {
		secretLog.Info("secret doesn't exist, creating it")

		if err := r.createKubernetesSecret(ctx, instance, *client); err != nil {
			log.Error(err, "could not create kubernetes secret")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil

	} else if err != nil {
		secretLog.Error(err, "could not get kubernetes secret")
		return ctrl.Result{}, fmt.Errorf("get secret: %w", err)
	}

	update := false
	if !areClientCredentialsCorrect(secret, *client) {
		secretLog.Info("secret data diverges from wanted, updating with correct values")
		if _, ok := secret.Data[cookieSecretKey]; !ok {
			cookieSecret, err := generateCookieSecret()
			if err != nil {
				return ctrl.Result{}, err
			}
			secret.Data[cookieSecretKey] = []byte(cookieSecret)
		}
		secret.Data[clientIdKey] = []byte(client.ClientID)
		secret.Data[clientSecretKey] = []byte(client.ClientSecret)
		update = true
	}

	if !controllerutil.HasControllerReference(secret) {
		if err := controllerutil.SetControllerReference(instance, secret, r.Scheme); err != nil {
			log.Error(err, "could not set controller reference")
			return ctrl.Result{}, err
		}
		update = true
	}

	if update {
		return ctrl.Result{}, r.Update(ctx, secret)
	}

	return ctrl.Result{}, nil
}

func (r *SimpleProxyClientReconciler) deleteKeycloakClient(ctx context.Context, clientId string) error {
	if err := r.oidcService.DeleteClient(ctx, DeleteClientRequest{
		Name: clientId,
	}); !errors.Is(err, ErrNotFound) {
		return err
	}

	return nil
}

func (r *SimpleProxyClientReconciler) createKubernetesSecret(ctx context.Context, instance *daplav1alpha1.SimpleProxyClient, client Client) error {
	cookieSecret, err := generateCookieSecret()
	if err != nil {
		return err
	}
	secret := corev1.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Name:        instance.Spec.SecretName,
			Namespace:   instance.Namespace,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
		Data: map[string][]byte{
			cookieSecretKey: []byte(cookieSecret),
			clientIdKey:     []byte(client.ClientID),
			clientSecretKey: []byte(client.ClientSecret),
		},
	}
	if err = controllerutil.SetControllerReference(instance, &secret, r.Scheme); err != nil {
		return err
	}
	return r.Create(ctx, &secret)
}

// areClientCredentialsCorrect reports whether the client ID and secret are correct
func areClientCredentialsCorrect(have *corev1.Secret, client Client) bool {
	if clientId, ok := have.Data[clientIdKey]; !ok || !bytes.Equal(clientId, []byte(client.ClientID)) {
		return false
	}
	if clientSecret, ok := have.Data[clientSecretKey]; !ok || !bytes.Equal(clientSecret, []byte(client.ClientSecret)) {
		return false
	}
	return true
}

func toClientId(name, namespace string) string {
	const clientIdFormat = "%s-%s"
	return fmt.Sprintf(clientIdFormat, namespace, name)
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
