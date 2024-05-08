// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
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

func rayNodeConfigUnitTests() {
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
					VMClass: "vm_class",
					Nfs:     "nfs",
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
	})
}
