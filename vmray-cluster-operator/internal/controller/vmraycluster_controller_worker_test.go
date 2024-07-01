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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func rayWorkerUnitTests() {
	Describe("VMRayCluster Controller Worker Tests", func() {

		var (
			nodeconfigName = "test-vm-ray-nodeconfig"
			testobjectname = "test-object"
			namespace      = "default"
		)

		Context("When reconciling a resource", func() {
			ctx := context.Background()

			typeNodeConfigNamespacedName := getNamespacedName(namespace, nodeconfigName)
			rayClusterNamespacedName := getNamespacedName(namespace, "test-vm-ray-cluster")

			BeforeEach(func() {

				createVmRayNodeConfig(ctx, suite.GetK8sClient(), namespace, nodeconfigName, testobjectname)

				vmraycluster := &vmrayv1alpha1.VMRayCluster{}
				err := suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, vmraycluster)
				if err != nil && client.IgnoreNotFound(err) == nil {
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
							Name:      rayClusterNamespacedName.Name,
							Namespace: namespace,
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
					Expect(suite.GetK8sClient().Create(ctx, vmrayclusterinstance)).To(Succeed())
				}
			})
			AfterEach(func() {
				deleteVmRayNodeConfig(ctx, suite.GetK8sClient(), namespace, nodeconfigName, testobjectname)
			})

			It("Life cycle of worker Node VM", func() {
				instance := &vmrayv1alpha1.VMRayCluster{}
				err := suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
				Expect(err).NotTo(HaveOccurred())
				provider := mockvmpv.NewMockVmProvider()
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

				provider.DeploySetResponse(1, nil)
				// first reconcile to deploy head node and set vm_status to initialized
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: rayClusterNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
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
					NamespacedName: rayClusterNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
				Expect(err).To(BeNil())

				reqFetchVMStatus := provider.FetchVmStatusGetRequest(1)
				name := instance.ObjectMeta.Name + "-h-" + instance.ObjectMeta.Labels[vmraycontroller.HeadNodeNounceLabel]
				Expect(reqFetchVMStatus.Name).Should(Equal(name))
				Expect(reqFetchVMStatus.Namespace).Should(Equal(instance.Namespace))

				Expect(instance.Status.HeadNodeStatus.VmStatus).Should(Equal(vmrayv1alpha1.RUNNING))
				Expect(instance.Status.HeadNodeStatus.RayStatus).Should(Equal(vmrayv1alpha1.RAY_INITIALIZED))

				// 3rd reconcile to set ray process status to running and mark the cluster state as healthy
				provider.DeploySetResponse(2, nil)
				provider.FetchVmStatusSetResponse(2, &instance.Status.HeadNodeStatus, nil)
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: rayClusterNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
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
					NamespacedName: rayClusterNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
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
					NamespacedName: rayClusterNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
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
				err := suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
				Expect(err).NotTo(HaveOccurred())
				provider := mockvmpv.NewMockVmProvider()
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

				status := vmrayv1alpha1.VMRayNodeStatus{
					Ip:        "",
					VmStatus:  vmrayv1alpha1.RUNNING,
					RayStatus: vmrayv1alpha1.RAY_RUNNING,
				}
				provider.FetchVmStatusSetResponse(1, &instance.Status.HeadNodeStatus, nil)
				provider.FetchVmStatusSetResponse(2, &status, nil)
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: rayClusterNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
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
				err := suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
				Expect(err).NotTo(HaveOccurred())
				provider := mockvmpv.NewMockVmProvider()
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

				headnodestatus := vmrayv1alpha1.VMRayNodeStatus{
					Ip:        "12.12.12.12",
					VmStatus:  vmrayv1alpha1.RUNNING,
					RayStatus: vmrayv1alpha1.RAY_INITIALIZED,
				}
				provider.FetchVmStatusSetResponse(1, &headnodestatus, nil)
				provider.DeploySetResponse(2, nil)

				// delete the worker node config here
				nodeconfiginstance := &vmrayv1alpha1.VMRayNodeConfig{}
				err = suite.GetK8sClient().Get(ctx, typeNodeConfigNamespacedName, nodeconfiginstance)
				Expect(err).NotTo(HaveOccurred())
				Expect(suite.GetK8sClient().Delete(ctx, nodeconfiginstance)).To(Succeed())

				// Failure to deploy worker node should mark the cluster unhealthy
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: rayClusterNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				err = suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
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
						Namespace: namespace,
					},
					Spec: vmrayv1alpha1.VMRayNodeConfigSpec{
						StorageClass:       testobjectname,
						VMClass:            testobjectname,
						VMImage:            testobjectname,
						VMPasswordSaltHash: "$6$test1234$9/BUZHNkvq.c1miDDMG5cHLmM4V7gbYdGuF0//3gSIh//DOyi7ypPCs6EAA9b8/tidHottL6UG0tG/RqTgAAi/",
						VMUser:             "ray-vm",
					},
				}
				Expect(suite.GetK8sClient().Create(ctx, nodeconfiginstance)).To(Succeed())
			})

			// negative test: failure to delete worker node
			It("should fail raycluster deletion if worker node fails to delete", func() {
				instance := &vmrayv1alpha1.VMRayCluster{}
				err := suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
				Expect(err).NotTo(HaveOccurred())
				provider := mockvmpv.NewMockVmProvider()
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

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
				deleteRayCluster(ctx, suite.GetK8sClient(), rayClusterNamespacedName, instance)
				// call reconciler to delete the cluster
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: rayClusterNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				err = suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Reason).Should(Equal(vmrayv1alpha1.FailureToDeleteWorkerNodeReason))
				Expect(instance.Status.Conditions[len(instance.Status.Conditions)-1].Type).Should(Equal(vmrayv1alpha1.VMRayClusterConditionClusterDelete))

			})

			// Validate worker node deletion
			It("should delete the worker nodes successfully", func() {
				instance := &vmrayv1alpha1.VMRayCluster{}
				err := suite.GetK8sClient().Get(ctx, rayClusterNamespacedName, instance)
				Expect(err).NotTo(HaveOccurred())
				provider := mockvmpv.NewMockVmProvider()
				controllerReconciler := vmraycontroller.NewVMRayClusterReconciler(suite.GetK8sClient(), suite.GetK8sClient().Scheme(), provider)

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
				deleteRayCluster(ctx, suite.GetK8sClient(), rayClusterNamespacedName, instance)
				// call reconciler to delete the cluster
				_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: rayClusterNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

		})
	})
}
