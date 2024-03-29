// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var longStr = strings.Repeat("longstr", 12)

func unitTests() {
	var (
		rayNodeConfig VMRayNodeConfig
	)
	Describe("VMRayNodeConfig validation webhook", func() {

		BeforeEach(func() {
			rayNodeConfig = VMRayNodeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "name",
				},
				Spec: VMRayNodeConfigSpec{
					VMClass:         "vm_class",
					ContentLibrary:  "content_library",
					Ovf:             "ovf",
					Nfs:             "nfs",
					StoragePolicy:   "storage_policy",
					NetworkPolicy:   "network_policy",
					CloudInitConfig: "cloud-init-config",
				},
			}
		})

		Context("when name is invalid", func() {

			It("should return error", func() {
				rayNodeConfig.Name = "invalid.name"

				err := suite.GetK8sClient().Create(context.TODO(), &rayNodeConfig)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("VMRayNodeConfig.vmray \"invalid.name\" is invalid"))
			})
		})

		Context("invalid vm_class due to non DNS complaint value", func() {

			It("should return error", func() {
				rayNodeConfig.Spec.VMClass = "test#"

				err := suite.GetK8sClient().Create(context.TODO(), &rayNodeConfig)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.vm_class: Invalid value: \"test#\""))
				Expect(err.Error()).To(ContainSubstring("Must be DNS complaint"))
			})
		})

		Context("invalid vm_class due to long value", func() {

			It("should return error", func() {
				rayNodeConfig.Spec.VMClass = longStr

				err := suite.GetK8sClient().Create(context.TODO(), &rayNodeConfig)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.vm_class: Invalid value"))
				Expect(err.Error()).To(ContainSubstring("Maximum 63 characters are allowed"))
			})
		})

		Context("invalid content_library due to long value", func() {

			It("should return error", func() {
				rayNodeConfig.Spec.ContentLibrary = longStr

				err := suite.GetK8sClient().Create(context.TODO(), &rayNodeConfig)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.content_library: Invalid value"))
				Expect(err.Error()).To(ContainSubstring("Maximum 80 characters are allowed."))
			})
		})

		Context("invalid ovf due to long value", func() {

			It("should return error", func() {
				rayNodeConfig.Spec.Ovf = longStr

				err := suite.GetK8sClient().Create(context.TODO(), &rayNodeConfig)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.ovf: Invalid value"))
				Expect(err.Error()).To(ContainSubstring("Maximum 80 characters are allowed."))
			})
		})

		Context("invalid storage_policy due to long value", func() {

			It("should return error", func() {
				rayNodeConfig.Spec.StoragePolicy = longStr

				err := suite.GetK8sClient().Create(context.TODO(), &rayNodeConfig)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.storage_policy: Invalid value"))
				Expect(err.Error()).To(ContainSubstring("Maximum 80 characters are allowed."))
			})
		})

		Context("invalid network_policy due to long value", func() {

			It("should return error", func() {
				rayNodeConfig.Spec.NetworkPolicy = longStr

				err := suite.GetK8sClient().Create(context.TODO(), &rayNodeConfig)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.network_policy: Invalid value"))
				Expect(err.Error()).To(ContainSubstring("Maximum 80 characters are allowed."))
			})
		})

		Context("invalid cloud_init_config due to non-DNS complain name", func() {

			It("should return error", func() {
				rayNodeConfig.Spec.CloudInitConfig = "test#"

				err := suite.GetK8sClient().Create(context.TODO(), &rayNodeConfig)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.cloud_init_config: Invalid value"))
				Expect(err.Error()).To(ContainSubstring("Must be DNS complaint"))
			})
		})
	})
}
