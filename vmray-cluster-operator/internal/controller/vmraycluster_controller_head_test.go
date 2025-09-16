// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vmrayv1alpha1 "github.com/vmware/ray-on-vcf/vmray-cluster-operator/api/v1alpha1"
	vmraycontroller "github.com/vmware/ray-on-vcf/vmray-cluster-operator/internal/controller"
	mockvmpv "github.com/vmware/ray-on-vcf/vmray-cluster-operator/pkg/provider/mock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testutil "github.com/vmware/ray-on-vcf/vmray-cluster-operator/test/builder/utils"
)

func rayHeadUnitTests() {
	Describe("VMRayCluster controller head tests", func() {

		var (
			testobjectname string = "test-object"
			namespace      string = "default"
		)

		Context("When reconciling a resource", func() {
			ctx := context.Background()

			BeforeEach(func() {
				testutil.CreateAuxiliaryDependencies(ctx, suite.GetK8sClient(), namespace, testobjectname)
			})
			AfterEach(func() {
				testutil.DeleteAuxiliaryDependencies(ctx, suite.GetK8sClient(), namespace, testobjectname)
			})

			// negative test: Invalid storage class, vm class & VM image.
			It("Negatve testing: check status conditions for invalid vm class, vmi & storage class", func() {

				provider := mockvmpv.NewMockVmProvider()
				namespacedName := testutil.GetNamespacedName(namespace, "vmrayclustertest-test")
				_ = testutil.CreateRayClusterInstance(ctx, suite.GetK8sClient(), namespace, "vmrayclustertest-test", "not-created")

				// Run a nodeconfig reconclie loop and make sure it is valid.
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)
				_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
				Expect(err).NotTo(HaveOccurred())

				instance := &vmrayv1alpha1.VMRayCluster{}
				err = suite.GetK8sClient().Get(ctx, namespacedName, instance)
				Expect(err).ToNot(HaveOccurred())

				Expect(instance.Status.Conditions[0].Type).To(Equal("InvalidVirtualMachineImage"))
				Expect(instance.Status.Conditions[0].Reason).To(Equal("ResourceNotFound"))
				Expect(instance.Status.Conditions[0].Message).To(Equal("virtualmachineimages.vmoperator.vmware.com \"not-created\" not found"))

				Expect(instance.Status.Conditions[1].Type).To(Equal("InvalidStorageClass"))
				Expect(instance.Status.Conditions[1].Reason).To(Equal("ResourceNotFound"))
				Expect(instance.Status.Conditions[1].Message).To(Equal("storageclasses.storage.k8s.io \"not-created\" not found"))

				Expect(instance.Status.Conditions[2].Type).To(Equal("InvalidVirtualMachineClass"))
				Expect(instance.Status.Conditions[2].Reason).To(Equal("ResourceNotFound"))
				Expect(instance.Status.Conditions[2].Message).To(Equal("virtualmachineclasses.vmoperator.vmware.com \"not-created\" not found"))
			})

			It("Life cycle of the head node VM, Ray Process and Ray Cluster Status update", func() {
				provider := mockvmpv.NewMockVmProvider()
				typeNamespacedName := testutil.GetNamespacedName(namespace, "vmrayclustertest3")
				instance := testutil.CreateRayClusterInstance(ctx, suite.GetK8sClient(), namespace, "vmrayclustertest3", testobjectname)

				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

				provider.DeployVmServiceSetResponse(1, "192.10.10.1", nil)
				provider.DeploySetResponse(1, nil)
				// first reconcile to deploy head node and set vm_status to initialized
				_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).ToNot(HaveOccurred())

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
				Expect(err).ToNot(HaveOccurred())

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
				Expect(err).ToNot(HaveOccurred())

				reqFetchVMStatus = provider.FetchVmStatusGetRequest(2)
				Expect(instance.Status.HeadNodeStatus.RayStatus).Should(Equal(vmrayv1alpha1.RAY_RUNNING))
				Expect(instance.Status.ClusterState).Should(Equal(vmrayv1alpha1.HEALTHY))

				provider.DeleteAuxiliaryResourcesSetResponse(1, nil)
				provider.DeleteSetResponse(1, nil)

				testutil.DeleteRayCluster(ctx, suite.GetK8sClient(), typeNamespacedName, instance)

				// call reconciler to delete the cluster
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

			})
			// negative test case
			It("Ray Cluster Get fails with instance not found error", func() {
				provider := mockvmpv.NewMockVmProvider()
				typeNamespacedName := testutil.GetNamespacedName(namespace, "vmrayclustertest5")
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)
				_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			// negative test case
			It("Ray Cluster delete fails with failure to delete auxiliary resource", func() {
				provider := mockvmpv.NewMockVmProvider()
				typeNamespacedName := testutil.GetNamespacedName(namespace, "vmrayclustertest6")
				instance := testutil.CreateRayClusterInstance(ctx, suite.GetK8sClient(), namespace, "vmrayclustertest6", testobjectname)
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

				provider.DeployVmServiceSetResponse(1, "192.10.10.1", nil)
				provider.DeploySetResponse(1, nil)
				_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = fmt.Errorf("Failure when trying to delete auxiliary resources for %s", instance.Name)
				provider.DeleteAuxiliaryResourcesSetResponse(1, err)
				testutil.DeleteRayCluster(ctx, suite.GetK8sClient(), typeNamespacedName, instance)
				// Call reconciler to delete the cluster.
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).ToNot(HaveOccurred())

				Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Reason).Should(Equal(vmrayv1alpha1.FailureToDeleteAuxiliaryResourcesReason))
				Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type).Should(Equal(vmrayv1alpha1.VMRayClusterConditionClusterDelete))

			})

			It("Ray Cluster delete fails with failure to delete head Node", func() {
				provider := mockvmpv.NewMockVmProvider()
				typeNamespacedName := testutil.GetNamespacedName(namespace, "vmrayclustertest7")
				instance := testutil.CreateRayClusterInstance(ctx, suite.GetK8sClient(), namespace, "vmrayclustertest7", testobjectname)
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

				provider.DeployVmServiceSetResponse(1, "192.10.10.1", nil)
				provider.DeploySetResponse(1, nil)
				_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = fmt.Errorf("Failure when trying to delete worker nodes %s", instance.Name)
				provider.DeleteAuxiliaryResourcesSetResponse(1, nil)
				provider.DeleteSetResponse(1, err)
				testutil.DeleteRayCluster(ctx, suite.GetK8sClient(), typeNamespacedName, instance)
				// call reconciler to delete the cluster
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).ToNot(HaveOccurred())

				Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Reason).Should(Equal(vmrayv1alpha1.FailureToDeleteHeadNodeReason))
				Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type).Should(Equal(vmrayv1alpha1.VMRayClusterConditionClusterDelete))
			})

			// negative test-cases
			It("Mark the Head Node state as failed if it looses IP", func() {
				provider := mockvmpv.NewMockVmProvider()
				typeNamespacedName := testutil.GetNamespacedName(namespace, "vmrayclustertest4")
				instance := testutil.CreateRayClusterInstance(ctx, suite.GetK8sClient(), namespace, "vmrayclustertest4", testobjectname)

				origInstance := instance.DeepCopy()

				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)
				err := suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).ToNot(HaveOccurred())

				// Update status of head node to running, but IP is not set.
				instance.Status.HeadNodeStatus.Ip = ""
				instance.Status.HeadNodeStatus.VmStatus = vmrayv1alpha1.RUNNING

				patch := client.MergeFrom(origInstance)
				err = suite.GetK8sClient().Status().Patch(ctx, instance, patch)
				Expect(err).ToNot(HaveOccurred())

				err = fmt.Errorf("Primary IPv4 not found for %s Node", instance.Name)
				provider.FetchVmStatusSetResponse(1, &instance.Status.HeadNodeStatus, err)

				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, typeNamespacedName, instance)
				Expect(err).ToNot(HaveOccurred())

				reqFetchVMStatus := provider.FetchVmStatusGetRequest(1)
				name := instance.ObjectMeta.Name + "-h-" + instance.ObjectMeta.Labels[vmraycontroller.HeadNodeNounceLabel]
				Expect(reqFetchVMStatus.Name).Should(Equal(name))
				Expect(reqFetchVMStatus.Namespace).Should(Equal(instance.Namespace))

				Expect(instance.Status.HeadNodeStatus.VmStatus).Should(Equal(vmrayv1alpha1.FAIL))
				Expect(instance.Status.HeadNodeStatus.RayStatus).Should(Equal(vmrayv1alpha1.RAY_FAIL))
				Expect(instance.Status.ClusterState).Should(Equal(vmrayv1alpha1.UNHEALTHY))
				testutil.DeleteRayCluster(ctx, suite.GetK8sClient(), typeNamespacedName, instance)
			})

		})
	})
}
