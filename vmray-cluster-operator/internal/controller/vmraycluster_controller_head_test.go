// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	vmraycontroller "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/internal/controller"
	mockvmpv "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/mock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func rayHeadUnitTests() {
	Describe("VMRayCluster controller head tests", func() {

		var (
			nodeconfigName string = "test-vm-ray-nodeconfig"
			testobjectname string = "test-object"
			namespace      string = "default"
		)

		Context("When reconciling a resource", func() {
			ctx := context.Background()

			BeforeEach(func() {
				createVmRayNodeConfig(ctx, suite.GetK8sClient(), namespace, nodeconfigName, testobjectname)
			})
			AfterEach(func() {
				deleteVmRayNodeConfig(ctx, suite.GetK8sClient(), namespace, nodeconfigName, testobjectname)
			})

			It("Life cycle of the head node VM, Ray Process and Ray Cluster Status update", func() {
				provider := mockvmpv.NewMockVmProvider()
				typeNamespacedName := getNamespacedName(namespace, "vmrayclustertest3")
				instance := createRayClusterInstance(ctx, suite.GetK8sClient(), namespace, "vmrayclustertest3", nodeconfigName)

				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

				provider.DeploySetResponse(1, nil)
				// first reconcile to deploy head node and set vm_status to initialized
				_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).To(BeNil())

				req_deploy := provider.DeployGetRequest(1)
				Expect(req_deploy.ClusterName).Should(Equal(instance.Name))
				Expect(req_deploy.Namespace).Should(Equal(instance.Namespace))
				Expect(instance.Status.HeadNodeStatus.VmStatus).Should(Equal(vmrayv1alpha1.INITIALIZED))

				// Assign IP here
				instance.Status.HeadNodeStatus.Ip = "12.12.12.12"
				provider.FetchVmStatusSetResponse(1, &instance.Status.HeadNodeStatus, nil)

				// 2nd reconcile to set the vm status to running and ray status to initialized
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).To(BeNil())

				reqFetchVMStatus := provider.FetchVmStatusGetRequest(1)
				name := instance.ObjectMeta.Name + "-h-" + instance.ObjectMeta.Labels[vmraycontroller.HeadNodeNounceLabel]
				Expect(reqFetchVMStatus.Name).Should(Equal(name))
				Expect(reqFetchVMStatus.Namespace).Should(Equal(instance.Namespace))

				Expect(instance.Status.HeadNodeStatus.VmStatus).Should(Equal(vmrayv1alpha1.RUNNING))
				Expect(instance.Status.HeadNodeStatus.RayStatus).Should(Equal(vmrayv1alpha1.RAY_INITIALIZED))

				// 3rd reconcile to set ray process status to running and mark the cluster state as healthy
				provider.FetchVmStatusSetResponse(2, &instance.Status.HeadNodeStatus, nil)
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).To(BeNil())

				reqFetchVMStatus = provider.FetchVmStatusGetRequest(2)
				Expect(instance.Status.HeadNodeStatus.RayStatus).Should(Equal(vmrayv1alpha1.RAY_RUNNING))
				Expect(instance.Status.ClusterState).Should(Equal(vmrayv1alpha1.HEALTHY))

				provider.DeleteAuxiliaryResourcesSetResponse(1, nil)
				provider.DeleteSetResponse(1, nil)

				deleteRayCluster(ctx, suite.GetK8sClient(), typeNamespacedName, instance)

				// call reconciler to delete the cluster
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

			})
			// negative test case
			It("Ray Cluster Get fails with instance not found error", func() {
				provider := mockvmpv.NewMockVmProvider()
				typeNamespacedName := getNamespacedName(namespace, "vmrayclustertest5")
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)
				_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			// negative test case
			It("Ray Cluster delete fails with failure to delete auxiliary resource", func() {
				provider := mockvmpv.NewMockVmProvider()
				typeNamespacedName := getNamespacedName(namespace, "vmrayclustertest6")
				instance := createRayClusterInstance(ctx, suite.GetK8sClient(), namespace, "vmrayclustertest6", nodeconfigName)
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

				provider.DeploySetResponse(1, nil)
				_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = fmt.Errorf("Failure when trying to delete auxiliary resources for %s", instance.Name)
				provider.DeleteAuxiliaryResourcesSetResponse(1, err)
				deleteRayCluster(ctx, suite.GetK8sClient(), typeNamespacedName, instance)
				//Call reconciler to delete the cluster
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).To(BeNil())

				Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Reason).Should(Equal(vmrayv1alpha1.FailureToDeleteAuxiliaryResourcesReason))
				Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type).Should(Equal(vmrayv1alpha1.VMRayClusterConditionClusterDelete))

			})

			It("Ray Cluster delete fails with failure to delete head Node", func() {
				provider := mockvmpv.NewMockVmProvider()
				typeNamespacedName := getNamespacedName(namespace, "vmrayclustertest7")
				instance := createRayClusterInstance(ctx, suite.GetK8sClient(), namespace, "vmrayclustertest7", nodeconfigName)
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

				provider.DeploySetResponse(1, nil)
				_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = fmt.Errorf("Failure when trying to delete worker nodes %s", instance.Name)
				provider.DeleteAuxiliaryResourcesSetResponse(1, nil)
				provider.DeleteSetResponse(1, err)
				deleteRayCluster(ctx, suite.GetK8sClient(), typeNamespacedName, instance)
				// call reconciler to delete the cluster
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).To(BeNil())

				Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Reason).Should(Equal(vmrayv1alpha1.FailureToDeleteHeadNodeReason))
				Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type).Should(Equal(vmrayv1alpha1.VMRayClusterConditionClusterDelete))
			})

			// negative test-cases
			It("Mark the Head Node state as failed if it looses IP", func() {
				provider := mockvmpv.NewMockVmProvider()
				typeNamespacedName := getNamespacedName(namespace, "vmrayclustertest4")
				instance := createRayClusterInstance(ctx, suite.GetK8sClient(), namespace, "vmrayclustertest4", nodeconfigName)

				origInstance := instance.DeepCopy()

				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)
				err := suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).To(BeNil())

				// Update status of head node to running, but IP is not set.
				instance.Status.HeadNodeStatus.Ip = ""
				instance.Status.HeadNodeStatus.VmStatus = vmrayv1alpha1.RUNNING

				patch := client.MergeFrom(origInstance)
				err = suite.GetK8sClient().Status().Patch(ctx, instance, patch)
				Expect(err).To(BeNil())

				err = fmt.Errorf("Primary IPv4 not found for %s Node", instance.Name)
				provider.FetchVmStatusSetResponse(1, &instance.Status.HeadNodeStatus, err)

				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).To(BeNil())

				reqFetchVMStatus := provider.FetchVmStatusGetRequest(1)
				name := instance.ObjectMeta.Name + "-h-" + instance.ObjectMeta.Labels[vmraycontroller.HeadNodeNounceLabel]
				Expect(reqFetchVMStatus.Name).Should(Equal(name))
				Expect(reqFetchVMStatus.Namespace).Should(Equal(instance.Namespace))

				Expect(instance.Status.HeadNodeStatus.VmStatus).Should(Equal(vmrayv1alpha1.FAIL))
				Expect(instance.Status.HeadNodeStatus.RayStatus).Should(Equal(vmrayv1alpha1.RAY_FAIL))
				Expect(instance.Status.ClusterState).Should(Equal(vmrayv1alpha1.UNHEALTHY))
				deleteRayCluster(ctx, suite.GetK8sClient(), typeNamespacedName, instance)
			})

		})
	})
}
