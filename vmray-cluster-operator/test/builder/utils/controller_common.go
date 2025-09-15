// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"context"

	. "github.com/onsi/gomega"
	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmrayv1alpha1 "github.com/vmware/ray-on-vcf/vmray-cluster-operator/api/v1alpha1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateRayClusterInstance(ctx context.Context, k8sClient client.Client,
	namespace, name, testobjectname string) *vmrayv1alpha1.VMRayCluster {
	port := uint(6379)
	head_node := vmrayv1alpha1.HeadNodeConfig{
		Port:     &port,
		NodeType: "node.1",
	}
	node_config := vmrayv1alpha1.CommonNodeConfig{
		MaxWorkers:   2,
		VMImage:      testobjectname,
		StorageClass: testobjectname,
		NodeTypes: map[string]vmrayv1alpha1.NodeType{
			"worker_1": {
				VMClass:    testobjectname,
				MinWorkers: 3,
				MaxWorkers: 5,
				Resources: vmrayv1alpha1.NodeResource{
					CPU:    2,
					Memory: 1024,
				},
			},
			"node.1": {
				VMClass: testobjectname,
			},
		},
	}
	resource := &vmrayv1alpha1.VMRayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: vmrayv1alpha1.VMRayClusterSpec{
			Image:      "rayproject/ray:2.5.0",
			HeadNode:   head_node,
			NodeConfig: node_config,
		},
	}
	Expect(k8sClient.Create(ctx, resource)).To(Succeed())
	return resource
}

func GetNamespacedName(namespace, name string) types.NamespacedName {
	typeNamespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	return typeNamespacedName
}

func DeleteRayCluster(ctx context.Context, k8sClient client.Client,
	typeNamespacedName types.NamespacedName, vmraycluster *vmrayv1alpha1.VMRayCluster) {
	err := k8sClient.Get(ctx, typeNamespacedName, vmraycluster)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Delete(ctx, vmraycluster)).To(Succeed())
}

func CreateAuxiliaryDependencies(ctx context.Context, k8sClient client.Client, namespace, testObjectName string) {

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
}

func DeleteAuxiliaryDependencies(ctx context.Context, k8sClient client.Client, namespace, testObjectName string) {

	testobjectNamespacedName := GetNamespacedName(namespace, testObjectName)

	// Delete VMclass, storageclass & VMI.
	vmi := &vmopv1.VirtualMachineImage{}
	err := k8sClient.Get(ctx, testobjectNamespacedName, vmi)
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
