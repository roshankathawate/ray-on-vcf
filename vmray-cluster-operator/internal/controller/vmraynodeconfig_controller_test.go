// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	vmraycontroller "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/internal/controller"
	ctrl "sigs.k8s.io/controller-runtime"
)

func vmRayNodeConfigTest() {
	Describe("VMRayNodeConfig Controller Tests", func() {

		var (
			nodeconfigName = "test-vm-ray-nodeconfig"
			namespace      = "default"
		)

		Context("Validation check when reconciling nodecofig resource", func() {
			ctx := context.Background()

			// negative test: Invalid storage class, vm class & VM image.
			It("Negatve testing: check status conditions for invalid vm class, vmi & storage class", func() {

				k8sClient := suite.GetK8sClient()
				namespacedName := getNamespacedName(namespace, nodeconfigName)

				nodeconfiginstance := &vmrayv1alpha1.VMRayNodeConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      nodeconfigName,
						Namespace: namespace,
					},
					Spec: vmrayv1alpha1.VMRayNodeConfigSpec{
						StorageClass:       "storage-class-1",
						VMClass:            "vmclass-1",
						VMImage:            "vmi-randonuuid",
						VMPasswordSaltHash: "$6$test1234$9/BUZHNkvq.c1miDDMG5cHLmM4V7gbYdGuF0//3gSIh//DOyi7ypPCs6EAA9b8/tidHottL6UG0tG/RqTgAAi/",
						VMUser:             "ray-vm",
					},
				}
				Expect(k8sClient.Create(ctx, nodeconfiginstance)).To(Succeed())

				// Run a nodeconfig reconclie loop and make sure it is valid.
				nodeConfigReconciler := vmraycontroller.NewVMRayNodeConfigReconciler(k8sClient, k8sClient.Scheme())
				_, err := nodeConfigReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
				Expect(err).NotTo(HaveOccurred())

				instance := &vmrayv1alpha1.VMRayNodeConfig{}
				err = k8sClient.Get(ctx, namespacedName, instance)
				Expect(err).To(BeNil())

				Expect(instance.Status.Conditions[0].Type).To(Equal("InvalidVirtualMachineImage"))
				Expect(instance.Status.Conditions[0].Reason).To(Equal("ResourceNotFound"))
				Expect(instance.Status.Conditions[0].Message).To(Equal("virtualmachineimages.vmoperator.vmware.com \"vmi-randonuuid\" not found"))

				Expect(instance.Status.Conditions[1].Type).To(Equal("InvalidStorageClass"))
				Expect(instance.Status.Conditions[1].Reason).To(Equal("ResourceNotFound"))
				Expect(instance.Status.Conditions[1].Message).To(Equal("storageclasses.storage.k8s.io \"storage-class-1\" not found"))

				Expect(instance.Status.Conditions[2].Type).To(Equal("InvalidVirtualMachineClass"))
				Expect(instance.Status.Conditions[2].Reason).To(Equal("ResourceNotFound"))
				Expect(instance.Status.Conditions[2].Message).To(Equal("virtualmachineclasses.vmoperator.vmware.com \"vmclass-1\" not found"))
			})
		})
	})
}
