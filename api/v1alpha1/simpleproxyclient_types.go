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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Required
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SimpleProxyClientSpec defines the desired state of SimpleProxyClient
type SimpleProxyClientSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// The target Keycloak realm.
	Realm string `json:"realm"`

	// RedirectUris is the list of allowed redirect URIs (callback URIs) for this client.
	// +kubebuilder:validation:MinItems=1
	RedirectUris []string `json:"redirectUris"`

	// TargetSecret is the name of the secret to create containing the client ID, client secret and cookie secret.
	TargetSecret string `json:"targetSecret"`
}

// SimpleProxyClientStatus defines the observed state of SimpleProxyClient
type SimpleProxyClientStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SimpleProxyClient is the Schema for the simpleproxyclients API
type SimpleProxyClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SimpleProxyClientSpec   `json:"spec,omitempty"`
	Status SimpleProxyClientStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SimpleProxyClientList contains a list of SimpleProxyClient
type SimpleProxyClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SimpleProxyClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SimpleProxyClient{}, &SimpleProxyClientList{})
}
