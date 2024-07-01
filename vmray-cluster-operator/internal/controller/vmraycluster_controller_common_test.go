// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller_test

import (
	"context"

	. "github.com/onsi/gomega"
	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	vmraycontroller "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/internal/controller"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createRayClusterInstance(ctx context.Context, k8sClient client.Client, namespace, name, nodeconfigName string) *vmrayv1alpha1.VMRayCluster {
	port := uint(6379)
	head_node := vmrayv1alpha1.HeadNodeConfig{
		NodeConfigName: nodeconfigName,
		Port:           &port,
	}
	worker_node := vmrayv1alpha1.WorkerNodeConfig{
		NodeConfigName: nodeconfigName,
		MinWorkers:     1,
		MaxWorkers:     2,
	}
	resource := &vmrayv1alpha1.VMRayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: vmrayv1alpha1.VMRayClusterSpec{
			Image:      "rayproject/ray:2.5.0",
			HeadNode:   head_node,
			WorkerNode: worker_node,
		},
	}
	Expect(k8sClient.Create(ctx, resource)).To(Succeed())
	return resource
}

func getNamespacedName(namespace, name string) types.NamespacedName {
	typeNamespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	return typeNamespacedName
}

func deleteRayCluster(ctx context.Context, k8sClient client.Client, typeNamespacedName types.NamespacedName, vmraycluster *vmrayv1alpha1.VMRayCluster) {
	err := k8sClient.Get(ctx, typeNamespacedName, vmraycluster)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Delete(ctx, vmraycluster)).To(Succeed())
}

func createVmRayNodeConfig(ctx context.Context, k8sClient client.Client, namespace, nodeConfigName, testObjectName string) {

	metaObj := metav1.ObjectMeta{
		Name:      testObjectName,
		Namespace: namespace,
	}

	// Create vmi, storage class and vm class.
	vmclass := &vmopv1.VirtualMachineClass{
		ObjectMeta: metaObj,
		Spec:       vmopv1.VirtualMachineClassSpec{},
	}
	Expect(k8sClient.Create(ctx, vmclass)).To(Succeed())

	vmi := &vmopv1.VirtualMachineImage{
		ObjectMeta: metaObj,
		Spec:       vmopv1.VirtualMachineImageSpec{},
	}
	Expect(k8sClient.Create(ctx, vmi)).To(Succeed())

	stgclass := &storagev1.StorageClass{
		ObjectMeta:  metaObj,
		Provisioner: "kubernetes.io/ray-test",
	}
	Expect(k8sClient.Create(ctx, stgclass)).To(Succeed())

	nodeconfiginstance := &vmrayv1alpha1.VMRayNodeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeConfigName,
			Namespace: namespace,
		},
		Spec: vmrayv1alpha1.VMRayNodeConfigSpec{
			StorageClass:       testObjectName,
			VMClass:            testObjectName,
			VMImage:            testObjectName,
			VMPasswordSaltHash: "$6$test1234$9/BUZHNkvq.c1miDDMG5cHLmM4V7gbYdGuF0//3gSIh//DOyi7ypPCs6EAA9b8/tidHottL6UG0tG/RqTgAAi/",
			VMUser:             "ray-vm",
		},
	}
	Expect(k8sClient.Create(ctx, nodeconfiginstance)).To(Succeed())

	// Run a nodeconfig reconclie loop and make sure it is valid.
	nodeConfigReconciler := vmraycontroller.NewVMRayNodeConfigReconciler(k8sClient, k8sClient.Scheme())
	_, err := nodeConfigReconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      nodeConfigName,
			Namespace: namespace,
		},
	})
	Expect(err).NotTo(HaveOccurred())
}

func deleteVmRayNodeConfig(ctx context.Context, k8sClient client.Client, namespace, nodeConfigName, testObjectName string) {

	nodeConfigNamespacedName := getNamespacedName(namespace, nodeConfigName)
	testobjectNamespacedName := getNamespacedName(namespace, testObjectName)

	// clean up VMRayNodeConfig instance
	nodeconfiginstance := &vmrayv1alpha1.VMRayNodeConfig{}
	err := k8sClient.Get(ctx, nodeConfigNamespacedName, nodeconfiginstance)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Delete(ctx, nodeconfiginstance)).To(Succeed())

	// Delete VMclass, storageclass & VMI.
	vmi := &vmopv1.VirtualMachineImage{}
	err = k8sClient.Get(ctx, testobjectNamespacedName, vmi)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Delete(ctx, vmi)).To(Succeed())

	vmclass := &vmopv1.VirtualMachineClass{}
	err = k8sClient.Get(ctx, testobjectNamespacedName, vmclass)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Delete(ctx, vmclass)).To(Succeed())

	stgclass := &storagev1.StorageClass{}
	err = k8sClient.Get(ctx, testobjectNamespacedName, stgclass)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Delete(ctx, stgclass)).To(Succeed())
}
