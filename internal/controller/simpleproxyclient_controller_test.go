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
	"context"
	"time"

	"github.com/Nerzal/gocloak/v13"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/statisticsnorway/keycloakerator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("SimpleProxyClient Controller", func() {

	const (
		name         = "test-spc"
		namespace    = "default"
		realm        = "master"
		redirectUri  = "http://test"
		targetSecret = "my-secret"
		timeout      = time.Second * 10
		duration     = time.Second * 10
		interval     = time.Millisecond * 250
	)

	Context("When deleting a SimpleProxyClient", func() {
		It("Should succeed even when the Keycloak client is missing", func() {
			By("Creating a new SimpleProxyClient")
			ctx := context.Background()
			spc := &v1alpha1.SimpleProxyClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: v1alpha1.SimpleProxyClientSpec{
					RedirectUris: []string{redirectUri},
					SecretName:   targetSecret,
				},
			}
			Expect(k8sClient.Create(ctx, spc)).Should(Succeed())

			By("Deleting all Keycloak clients")
			kc.clients = []gocloak.Client{}

			lookup := types.NamespacedName{Name: name, Namespace: namespace}
			createdSPC := &v1alpha1.SimpleProxyClient{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookup, createdSPC)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Deleting the SimpleProxyClient")
			Expect(k8sClient.Delete(ctx, spc)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookup, createdSPC)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})

})
