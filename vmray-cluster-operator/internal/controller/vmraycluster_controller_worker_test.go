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

var _ = Describe("VMRayCluster Controller Worker Tests", func() {

	Context("When reconciling a resource", func() {
		ctx := context.Background()

		typeNodeConfigNamespacedName := types.NamespacedName{
			Name:      nodeconfigName,
			Namespace: "default",
		}
		typeNamespacedName := types.NamespacedName{
			Name:      "test-vm-ray-cluster",
			Namespace: "default",
		}

		vmraynodeconfig := &vmrayv1alpha1.VMRayNodeConfig{}
		vmraycluster := &vmrayv1alpha1.VMRayCluster{}
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
			err = k8sClient.Get(ctx, typeNamespacedName, vmraycluster)
			if err != nil && errors.IsNotFound(err) {
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
				desired_workers := []string{"worker1"}
				vmrayclusterinstance := &vmrayv1alpha1.VMRayCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      typeNamespacedName.Name,
						Namespace: "default",
					},
					Spec: vmrayv1alpha1.VMRayClusterSpec{
						Image:          "rayproject/ray:2.5.0",
						HeadNode:       head_node,
						WorkerNode:     worker_node,
						JupyterHub:     &jupyterhub,
						Monitoring:     &monitoring,
						DesiredWorkers: desired_workers,
					},
				}
				Expect(k8sClient.Create(ctx, vmrayclusterinstance)).To(Succeed())
			}
		})
		AfterEach(func() {
			// clean up VMRayNodeConfig instance
			nodeconfiginstance := &vmrayv1alpha1.VMRayNodeConfig{}
			err := k8sClient.Get(ctx, typeNodeConfigNamespacedName, nodeconfiginstance)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, nodeconfiginstance)).To(Succeed())
		})

		It("Life cycle of worker Node VM", func() {
			instance := &vmrayv1alpha1.VMRayCluster{}
			err := k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).NotTo(HaveOccurred())
			provider := mockvmpv.NewMockVmProvider()
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
			Expect(err).NotTo(HaveOccurred())

			reqdeploy := provider.DeployGetRequest(1)
			Expect(reqdeploy.ClusterName).Should(Equal(instance.Name))
			Expect(reqdeploy.Namespace).Should(Equal(instance.Namespace))
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
			name := instance.ObjectMeta.Name + "-h-" + instance.ObjectMeta.Labels[HeadNodeNounceLabel]
			Expect(reqFetchVMStatus.Name).Should(Equal(name))
			Expect(reqFetchVMStatus.Namespace).Should(Equal(instance.Namespace))

			Expect(instance.Status.HeadNodeStatus.VmStatus).Should(Equal(vmrayv1alpha1.RUNNING))
			Expect(instance.Status.HeadNodeStatus.RayStatus).Should(Equal(vmrayv1alpha1.RAY_INITIALIZED))

			// 3rd reconcile to set ray process status to running and mark the cluster state as healthy
			provider.DeploySetResponse(2, nil)
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
			reqdeploy = provider.DeployGetRequest(2)
			Expect(reqdeploy.ClusterName).Should(Equal(instance.Name))
			Expect(reqdeploy.Namespace).Should(Equal(instance.Namespace))
			Expect(instance.Status.CurrentWorkers["worker1"].VmStatus).Should(Equal(vmrayv1alpha1.INITIALIZED))

			// call reconcile to move worker VMstate to RUNNING
			status := vmrayv1alpha1.VMRayNodeStatus{
				Ip:       "12.12.12.12",
				VmStatus: vmrayv1alpha1.INITIALIZED,
			}

			provider.FetchVmStatusSetResponse(3, &instance.Status.HeadNodeStatus, nil)
			provider.FetchVmStatusSetResponse(4, &status, nil)

			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).To(BeNil())

			reqFetchVMStatus = provider.FetchVmStatusGetRequest(4)
			Expect(reqFetchVMStatus.Name).Should(Equal("worker1"))
			Expect(reqFetchVMStatus.Namespace).Should(Equal(instance.Namespace))
			Expect(instance.Status.CurrentWorkers["worker1"].VmStatus).Should(Equal(vmrayv1alpha1.RUNNING))
			Expect(instance.Status.CurrentWorkers["worker1"].RayStatus).Should(Equal(vmrayv1alpha1.RAY_INITIALIZED))

			// call reconcile to move worker Ray Status to RUNNING.
			status = vmrayv1alpha1.VMRayNodeStatus{
				Ip:        "12.12.12.12",
				VmStatus:  vmrayv1alpha1.RUNNING,
				RayStatus: vmrayv1alpha1.RAY_INITIALIZED,
			}
			provider.FetchVmStatusSetResponse(5, &instance.Status.HeadNodeStatus, nil)
			provider.FetchVmStatusSetResponse(6, &status, nil)
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).To(BeNil())

			reqFetchVMStatus = provider.FetchVmStatusGetRequest(6)

			Expect(reqFetchVMStatus.Name).Should(Equal("worker1"))
			Expect(reqFetchVMStatus.Namespace).Should(Equal(instance.Namespace))
			Expect(instance.Status.CurrentWorkers["worker1"].VmStatus).Should(Equal(vmrayv1alpha1.RUNNING))
			Expect(instance.Status.CurrentWorkers["worker1"].RayStatus).Should(Equal(vmrayv1alpha1.RAY_RUNNING))
		})
		// negative test: Losing worker node IP
		It("Worker node losing IP marks the VM and Ray status as failed", func() {
			instance := &vmrayv1alpha1.VMRayCluster{}
			err := k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).NotTo(HaveOccurred())
			provider := mockvmpv.NewMockVmProvider()
			controllerReconciler := &VMRayClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				provider: provider,
				nlcm:     lcm.NewNodeLifecycleManager(provider),
			}

			status := vmrayv1alpha1.VMRayNodeStatus{
				Ip:        "",
				VmStatus:  vmrayv1alpha1.RUNNING,
				RayStatus: vmrayv1alpha1.RAY_RUNNING,
			}
			provider.FetchVmStatusSetResponse(1, &instance.Status.HeadNodeStatus, nil)
			provider.FetchVmStatusSetResponse(2, &status, nil)
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).To(BeNil())

			reqFetchVMStatus := provider.FetchVmStatusGetRequest(2)

			Expect(reqFetchVMStatus.Name).Should(Equal("worker1"))
			Expect(reqFetchVMStatus.Namespace).Should(Equal(instance.Namespace))
			Expect(instance.Status.CurrentWorkers["worker1"].VmStatus).Should(Equal(vmrayv1alpha1.FAIL))
			Expect(instance.Status.CurrentWorkers["worker1"].RayStatus).Should(Equal(vmrayv1alpha1.RAY_FAIL))
		})

		// negative test: Missing worker node config
		It("should mark the cluster unhealthy if worker node deployment failed due to missing node config ", func() {
			instance := &vmrayv1alpha1.VMRayCluster{}
			err := k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).NotTo(HaveOccurred())
			provider := mockvmpv.NewMockVmProvider()
			controllerReconciler := &VMRayClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				provider: provider,
				nlcm:     lcm.NewNodeLifecycleManager(provider),
			}

			headnodestatus := vmrayv1alpha1.VMRayNodeStatus{
				Ip:        "12.12.12.12",
				VmStatus:  vmrayv1alpha1.RUNNING,
				RayStatus: vmrayv1alpha1.RAY_INITIALIZED,
			}
			provider.FetchVmStatusSetResponse(1, &headnodestatus, nil)
			provider.DeploySetResponse(2, nil)

			// delete the worker node config here
			nodeconfiginstance := &vmrayv1alpha1.VMRayNodeConfig{}
			err = k8sClient.Get(ctx, typeNodeConfigNamespacedName, nodeconfiginstance)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, nodeconfiginstance)).To(Succeed())

			// Failure to deploy worker node should mark the cluster unhealthy
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).NotTo(HaveOccurred())

			Expect(instance.Status.CurrentWorkers["worker1"].VmStatus).Should(Equal(vmrayv1alpha1.FAIL))
			Expect(instance.Status.CurrentWorkers["worker1"].RayStatus).Should(Equal(vmrayv1alpha1.RAY_FAIL))
			Expect(instance.Status.ClusterState).Should(Equal(vmrayv1alpha1.UNHEALTHY))
			Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Reason).Should(Equal(vmrayv1alpha1.FailureToDeployNodeReason))
			Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type).Should(Equal(vmrayv1alpha1.VMRayClusterConditionWorkerNodeReady))
			// create worker node config again here so that clean-up goes through
			nodeconfiginstance = &vmrayv1alpha1.VMRayNodeConfig{
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
		})

		// negative test: failure to delete worker node
		It("should fail raycluster deletion if worker node fails to delete", func() {
			instance := &vmrayv1alpha1.VMRayCluster{}
			err := k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).NotTo(HaveOccurred())
			provider := mockvmpv.NewMockVmProvider()
			controllerReconciler := &VMRayClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				provider: provider,
				nlcm:     lcm.NewNodeLifecycleManager(provider),
			}

			headNodeStatus := vmrayv1alpha1.VMRayNodeStatus{
				Ip:        "12.12.12.12",
				VmStatus:  vmrayv1alpha1.RUNNING,
				RayStatus: vmrayv1alpha1.RAY_RUNNING,
			}
			workerNodeStatus := vmrayv1alpha1.VMRayNodeStatus{
				Ip:        "12.12.12.12",
				VmStatus:  vmrayv1alpha1.RUNNING,
				RayStatus: vmrayv1alpha1.RAY_RUNNING,
			}
			provider.FetchVmStatusSetResponse(1, &headNodeStatus, nil)
			provider.FetchVmStatusSetResponse(2, &workerNodeStatus, nil)

			err = fmt.Errorf("Failure when trying to delete worker nodes. %s ", instance.Name)
			provider.DeleteAuxiliaryResourcesSetResponse(1, nil)
			provider.DeleteSetResponse(1, err)
			provider.DeleteSetResponse(2, nil)
			deleteRayCluster(ctx, typeNamespacedName, instance)
			// call reconciler to delete the cluster
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Reason).Should(Equal(vmrayv1alpha1.FailureToDeleteWorkerNodeReason))
			Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type).Should(Equal(vmrayv1alpha1.VMRayClusterConditionClusterDelete))

		})

		// Validate worker node deletion
		It("should delete the worker nodes successfully", func() {
			instance := &vmrayv1alpha1.VMRayCluster{}
			err := k8sClient.Get(ctx, typeNamespacedName, instance)
			Expect(err).NotTo(HaveOccurred())
			provider := mockvmpv.NewMockVmProvider()
			controllerReconciler := &VMRayClusterReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				provider: provider,
				nlcm:     lcm.NewNodeLifecycleManager(provider),
			}

			headNodeStatus := vmrayv1alpha1.VMRayNodeStatus{
				Ip:        "12.12.12.12",
				VmStatus:  vmrayv1alpha1.RUNNING,
				RayStatus: vmrayv1alpha1.RAY_RUNNING,
			}
			workerNodeStatus := vmrayv1alpha1.VMRayNodeStatus{
				Ip:        "12.12.12.12",
				VmStatus:  vmrayv1alpha1.RUNNING,
				RayStatus: vmrayv1alpha1.RAY_RUNNING,
			}
			provider.FetchVmStatusSetResponse(1, &headNodeStatus, nil)
			provider.FetchVmStatusSetResponse(2, &workerNodeStatus, nil)

			provider.DeleteAuxiliaryResourcesSetResponse(1, nil)
			provider.DeleteSetResponse(1, nil)
			deleteRayCluster(ctx, typeNamespacedName, instance)
			// call reconciler to delete the cluster
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})

	})
})
