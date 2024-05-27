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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SimpleProxyClient Webhook", func() {

	DescribeTable("When modyfing fields in SimpleProxyClient under Validating Webhook", func(oldSpec SimpleProxyClientSpec) {
		spc := &SimpleProxyClient{
			Spec: SimpleProxyClientSpec{
				Realm:        "test1",
				RedirectUris: []string{"http://localhost1"},
				TargetSecret: "test1",
			},
		}
		old := &SimpleProxyClient{Spec: oldSpec}
		Expect(spc.ValidateUpdate(old)).Error().Should(HaveOccurred())
	},
		Entry("When Realm is changed", SimpleProxyClientSpec{Realm: "test2", RedirectUris: []string{"http://localhost1"}, TargetSecret: "test1"}),
		Entry("When RedirectUris is changed", SimpleProxyClientSpec{Realm: "test1", RedirectUris: []string{"http://localhost2"}, TargetSecret: "test1"}),
		Entry("When TargetSecret is changed", SimpleProxyClientSpec{Realm: "test1", RedirectUris: []string{"http://localhost1"}, TargetSecret: "test2"}),
	)

	Context("When the wrong kind is sent to ValidateUpdate", func() {
		It("Should fail", func() {
			By("Sending in a nil object")
			spc := &SimpleProxyClient{}
			Expect(spc.ValidateUpdate(nil)).Error().Should(HaveOccurred())
		})
	})


	Context("When validating a non-changed spec", func() {
		It("Should succeed", func() {
			By("Sending in the same object")
			spc := &SimpleProxyClient{}
			Expect(spc.ValidateUpdate(spc)).Error().ShouldNot(HaveOccurred())
		})
	})

	DescribeTable("When creating SimpleProxyClient under Validating Webhook", func(spec SimpleProxyClientSpec) {

		spc := &SimpleProxyClient{
			Spec: spec,
		}
		Expect(spc.ValidateCreate()).Error().ShouldNot(HaveOccurred())
	},
		Entry("When spec is empty", SimpleProxyClientSpec{}),
		Entry("When spec if filled", SimpleProxyClientSpec{Realm: "test", RedirectUris: []string{"http://test"}, TargetSecret: "test"}),
		Entry("When spec is missing redirect URIs", SimpleProxyClientSpec{Realm: "test", TargetSecret: "test"}),
	)

	DescribeTable("When deleting SimpleProxyClient under Validating Webhook", func(spec SimpleProxyClientSpec) {

		spc := &SimpleProxyClient{
			Spec: spec,
		}
		Expect(spc.ValidateDelete()).Error().ShouldNot(HaveOccurred())
	},
		Entry("When spec is empty", SimpleProxyClientSpec{}),
		Entry("When spec if filled", SimpleProxyClientSpec{Realm: "test", RedirectUris: []string{"http://test"}, TargetSecret: "test"}),
		Entry("When spec is missing redirect URIs", SimpleProxyClientSpec{Realm: "test", TargetSecret: "test"}),
	)

})
