// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/internal/controller/lcm"
	mockvmpv "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/mock"
)

const nodeconfigName = "test-vm-ray-nodeconfig"

func createRayClusterInstance(ctx context.Context, name string) *vmrayv1alpha1.VMRayCluster {
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
	jupyterhub := vmrayv1alpha1.JupyterHubConfig{
		Image:             "quay.io/jupyterhub/jupyterhub",
		DockerCredsSecret: "secret",
	}
	monitoring := vmrayv1alpha1.MonitoringConfig{
		PrometheusImage:   "prom/prometheus",
		GrafanaImage:      "grafana/grafana-oss",
		DockerCredsSecret: "secret",
	}
	resource := &vmrayv1alpha1.VMRayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: vmrayv1alpha1.VMRayClusterSpec{
			Image:      "rayproject/ray:2.5.0",
			HeadNode:   head_node,
			WorkerNode: worker_node,
			JupyterHub: &jupyterhub,
			Monitoring: &monitoring,
		},
	}
	Expect(k8sClient.Create(ctx, resource)).To(Succeed())
	return resource
}

func getTypeNamespacedName(name string) types.NamespacedName {
	typeNamespacedName := types.NamespacedName{
		Name:      name,
		Namespace: "default",
	}
	return typeNamespacedName
}

func deleteRayCluster(ctx context.Context, typeNamespacedName types.NamespacedName, vmraycluster *vmrayv1alpha1.VMRayCluster) {
	err := k8sClient.Get(ctx, typeNamespacedName, vmraycluster)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Delete(ctx, vmraycluster)).To(Succeed())
}

var _ = Describe("VMRayCluster Controller", func() {

	Context("When reconciling a resource", func() {
		ctx := context.Background()

		typeNodeConfigNamespacedName := types.NamespacedName{
			Name:      nodeconfigName,
			Namespace: "default",
		}

		vmraynodeconfig := &vmrayv1alpha1.VMRayNodeConfig{}
		BeforeEach(func() {
			err := k8sClient.Get(ctx, typeNodeConfigNamespacedName, vmraynodeconfig)
			if err != nil && errors.IsNotFound(err) {
				nodeconfiginstance := &vmrayv1alpha1.VMRayNodeConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      nodeconfigName,
						Namespace: "default",
					},
					Spec: vmrayv1alpha1.VMRayNodeConfigSpec{
						StorageClass:       "wcp-default-storage-profile",
						VMClass:            "best-effort-xlarge",
						VMImage:            "vmi-c446a19fe8559b14b",
						VMPasswordSaltHash: "$6$test1234$9/BUZHNkvq.c1miDDMG5cHLmM4V7gbYdGuF0//3gSIh//DOyi7ypPCs6EAA9b8/tidHottL6UG0tG/RqTgAAi/",
						VMUser:             "ray-vm",
					},
				}
				Expect(k8sClient.Create(ctx, nodeconfiginstance)).To(Succeed())
			}
		})
		AfterEach(func() {
			// clean up VMRayNodeConfig instance
			nodeconfiginstance := &vmrayv1alpha1.VMRayNodeConfig{}
			err := k8sClient.Get(ctx, typeNodeConfigNamespacedName, nodeconfiginstance)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, nodeconfiginstance)).To(Succeed())
		})

		It("Life cycle of the head node VM, Ray Process and Ray Cluster Status update", func() {
			provider := mockvmpv.NewMockVmProvider()
			typeNamespacedName := getTypeNamespacedName("vmrayclustertest3")
			instance := createRayClusterInstance(ctx, "vmrayclustertest3")

			controllerReconciler := &VMRayClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				provider: provider,
				nlcm:     lcm.NewNodeLifecycleManager(provider),
			}

			provider.DeploySetResponse(1, nil)
			// first reconcile to deploy head node and set vm_status to initialized
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).To(BeNil())

			req_deploy := provider.DeployGetRequest(1)
			Expect(req_deploy.ClusterName).Should(Equal(instance.Name))
			Expect(req_deploy.Namespace).Should(Equal(instance.Namespace))
			Expect(instance.Status.HeadNodeStatus.VmStatus).Should(Equal(vmrayv1alpha1.INITIALIZED))
			// assign IP here
			instance.Status.HeadNodeStatus.Ip = "12.12.12.12"
			provider.FetchVmStatusSetResponse(1, &instance.Status.HeadNodeStatus, nil)

			// 2nd reconcile to set the vm status to running and ray status to initialized
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).To(BeNil())

			reqFetchVMStatus := provider.FetchVmStatusGetRequest(1)
			Expect(reqFetchVMStatus.Name).Should(Equal(instance.Name + "-head"))
			Expect(reqFetchVMStatus.Namespace).Should(Equal(instance.Namespace))

			Expect(instance.Status.HeadNodeStatus.VmStatus).Should(Equal(vmrayv1alpha1.RUNNING))
			Expect(instance.Status.HeadNodeStatus.RayStatus).Should(Equal(vmrayv1alpha1.RAY_INITIALIZED))

			// 3rd reconcile to set ray process status to running and mark the cluster state as healthy
			provider.FetchVmStatusSetResponse(2, &instance.Status.HeadNodeStatus, nil)
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).To(BeNil())

			reqFetchVMStatus = provider.FetchVmStatusGetRequest(2)
			Expect(instance.Status.HeadNodeStatus.RayStatus).Should(Equal(vmrayv1alpha1.RAY_RUNNING))
			Expect(instance.Status.ClusterState).Should(Equal(vmrayv1alpha1.HEALTHY))

			provider.DeleteAuxiliaryResourcesSetResponse(1, nil)
			provider.DeleteSetResponse(1, nil)
			deleteRayCluster(ctx, typeNamespacedName, instance)
			// call reconciler to delete the cluster
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

		})
		// negative test case
		It("Ray Cluster Get fails with instance not found error", func() {
			provider := mockvmpv.NewMockVmProvider()
			typeNamespacedName := getTypeNamespacedName("vmrayclustertest5")
			controllerReconciler := &VMRayClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				provider: provider,
				nlcm:     lcm.NewNodeLifecycleManager(provider),
			}
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		// negative test case
		It("Ray Cluster delete fails with failure to delete auxiliary resource", func() {
			provider := mockvmpv.NewMockVmProvider()
			typeNamespacedName := getTypeNamespacedName("vmrayclustertest6")
			instance := createRayClusterInstance(ctx, "vmrayclustertest6")
			controllerReconciler := &VMRayClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				provider: provider,
				nlcm:     lcm.NewNodeLifecycleManager(provider),
			}

			provider.DeploySetResponse(1, nil)
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = fmt.Errorf("Failure when trying to delete auxiliary resources for %s", instance.Name)
			provider.DeleteAuxiliaryResourcesSetResponse(1, err)
			deleteRayCluster(ctx, typeNamespacedName, instance)
			//Call reconciler to delete the cluster
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).To(BeNil())

			Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Reason).Should(Equal(vmrayv1alpha1.FailureToDeleteAuxiliaryResourcesReason))
			Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type).Should(Equal(vmrayv1alpha1.VMRayClusterConditionClusterDelete))

		})

		It("Ray Cluster delete fails with failure to delete head Node", func() {
			provider := mockvmpv.NewMockVmProvider()
			typeNamespacedName := getTypeNamespacedName("vmrayclustertest7")
			instance := createRayClusterInstance(ctx, "vmrayclustertest7")
			controllerReconciler := &VMRayClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				provider: provider,
				nlcm:     lcm.NewNodeLifecycleManager(provider),
			}

			provider.DeploySetResponse(1, nil)
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = fmt.Errorf("Failure when trying to delete worker nodes %s", instance.Name)
			provider.DeleteAuxiliaryResourcesSetResponse(1, nil)
			provider.DeleteSetResponse(1, err)
			deleteRayCluster(ctx, typeNamespacedName, instance)
			// call reconciler to delete the cluster
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).To(BeNil())

			Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Reason).Should(Equal(vmrayv1alpha1.FailureToDeleteHeadNodeReason))
			Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type).Should(Equal(vmrayv1alpha1.VMRayClusterConditionClusterDelete))
		})

		// negative test-cases
		It("Mark the Head Node state as failed if it looses IP", func() {
			provider := mockvmpv.NewMockVmProvider()
			typeNamespacedName := getTypeNamespacedName("vmrayclustertest4")
			instance := createRayClusterInstance(ctx, "vmrayclustertest4")
			re := reconcileEnvelope{
				CurrentClusterState:  instance,
				OriginalClusterState: instance.DeepCopy(),
			}
			controllerReconciler := &VMRayClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				provider: provider,
				nlcm:     lcm.NewNodeLifecycleManager(provider),
			}
			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).To(BeNil())
			instance.Status.HeadNodeStatus.Ip = ""
			instance.Status.HeadNodeStatus.VmStatus = vmrayv1alpha1.RUNNING
			_, err = controllerReconciler.updateStatus(ctx, re)
			Expect(err).To(BeNil())
			err = fmt.Errorf("Primary IPv4 not found for %s Node", instance.Name)
			provider.FetchVmStatusSetResponse(1, &instance.Status.HeadNodeStatus, err)

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).To(BeNil())

			reqFetchVMStatus := provider.FetchVmStatusGetRequest(1)
			Expect(reqFetchVMStatus.Name).Should(Equal(instance.Name + "-head"))
			Expect(reqFetchVMStatus.Namespace).Should(Equal(instance.Namespace))

			Expect(instance.Status.HeadNodeStatus.VmStatus).Should(Equal(vmrayv1alpha1.FAIL))
			Expect(instance.Status.HeadNodeStatus.RayStatus).Should(Equal(vmrayv1alpha1.RAY_FAIL))
			Expect(instance.Status.ClusterState).Should(Equal(vmrayv1alpha1.UNHEALTHY))
			deleteRayCluster(ctx, typeNamespacedName, instance)
		})

	})
})
