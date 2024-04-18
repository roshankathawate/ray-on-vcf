// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func vmRayClusterUnitTests() {
	var (
		rayCluster VMRayCluster
	)
	Describe("VMRayCluster validating webhook", func() {

		BeforeEach(func() {
			head_node := HeadNodeConfig{
				NodeConfigName: "head_node",
			}
			worker_node := WorkerNodeConfig{
				NodeConfigName: "worker_node",
				MinWorkers:     0,
				MaxWorkers:     1,
			}
			jupyterhub := &JupyterHubConfig{
				Image:             "quay.io/jupyterhub/jupyterhub",
				DockerCredsSecret: "secret",
			}
			monitoring := &MonitoringConfig{
				PrometheusImage:   "prom/prometheus",
				GrafanaImage:      "grafana/grafana-oss",
				DockerCredsSecret: "secret",
			}
			rayCluster = VMRayCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "invalid.name",
				},
				Spec: VMRayClusterSpec{
					Image:      "rayproject/ray:2.5.0",
					HeadNode:   head_node,
					WorkerNode: worker_node,
					JupyterHub: jupyterhub,
					Monitoring: monitoring,
				},
			}
		})

		Context("when name is invalid", func() {
			It("should return error", func() {
				rayCluster.Name = "invalid.name"
				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("metadata.name: Invalid value: \"invalid.name\""))
			})
		})

		Context("invalid head_node config due to missing node config name", func() {

			It("should return error", func() {
				head_node := HeadNodeConfig{}
				worker_node := WorkerNodeConfig{
					NodeConfigName: "worker_node",
					MinWorkers:     0,
					MaxWorkers:     1,
				}
				jupyterhub := &JupyterHubConfig{
					Image:             "quay.io/jupyterhub/jupyterhub",
					DockerCredsSecret: "secret",
				}
				monitoring := &MonitoringConfig{
					PrometheusImage:   "prom/prometheus",
					GrafanaImage:      "grafana/grafana-oss",
					DockerCredsSecret: "secret",
				}
				VMRayCluster := VMRayCluster{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "name",
					},
					Spec: VMRayClusterSpec{
						Image:      "rayproject/ray:2.5.0",
						HeadNode:   head_node,
						WorkerNode: worker_node,
						JupyterHub: jupyterhub,
						Monitoring: monitoring,
					},
				}

				err := suite.GetK8sClient().Create(context.TODO(), &VMRayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.HeadNode.NodeConfig: Invalid value: \"NodeConfig\""))
				Expect(err.Error()).To(ContainSubstring("NodeConfig is a required field in HeadNodeConfig"))
			})
		})

		Context("invalid worker_node config due to missing node config name", func() {

			It("should return error", func() {
				head_node := HeadNodeConfig{
					NodeConfigName: "head_node",
				}
				worker_node := WorkerNodeConfig{
					MinWorkers: 0,
					MaxWorkers: 1,
				}
				jupyterhub := &JupyterHubConfig{
					Image:             "quay.io/jupyterhub/jupyterhub",
					DockerCredsSecret: "secret",
				}
				monitoring := &MonitoringConfig{
					PrometheusImage:   "prom/prometheus",
					GrafanaImage:      "grafana/grafana-oss",
					DockerCredsSecret: "secret",
				}
				VMRayCluster := VMRayCluster{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "name",
					},
					Spec: VMRayClusterSpec{
						Image:      "rayproject/ray:2.5.0",
						HeadNode:   head_node,
						WorkerNode: worker_node,
						JupyterHub: jupyterhub,
						Monitoring: monitoring,
					},
				}

				err := suite.GetK8sClient().Create(context.TODO(), &VMRayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.WorkerNode.NodeConfig: Invalid value: \"NodeConfig\""))
				Expect(err.Error()).To(ContainSubstring("NodeConfig is a required field in WorkerNodeConfig"))
			})
		})

		Context("invalid Min/Max workers", func() {

			It("should return error", func() {
				rayCluster.Spec.WorkerNode.MinWorkers = 4
				rayCluster.Spec.WorkerNode.MaxWorkers = 1

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.worker_node: Invalid value: \"min_workers/max_workers\""))
				Expect(err.Error()).To(ContainSubstring("Min workers cannot be more than Max workers"))
			})
		})

		Context("invalid Ray docker image", func() {

			It("should return error", func() {
				rayCluster.Spec.Image = "-"

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.image: Invalid value: \"-\""))
				Expect(err.Error()).To(ContainSubstring("Docker image is invalid"))
			})
		})

		Context("invalid jupyterhub docker image", func() {

			It("should return error", func() {
				rayCluster.Spec.JupyterHub.Image = "-"

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.jupyterhub.image: Invalid value: \"-\""))
				Expect(err.Error()).To(ContainSubstring("Docker image is invalid"))
			})
		})

		Context("invalid prometheus docker image", func() {

			It("should return error", func() {
				rayCluster.Spec.Monitoring.PrometheusImage = "-"

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.monitoring.prometheus_image: Invalid value: \"-\""))
				Expect(err.Error()).To(ContainSubstring("Docker image is invalid"))
			})
		})

		Context("invalid grafana image", func() {

			It("should return error", func() {
				rayCluster.Spec.Monitoring.GrafanaImage = "-"

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.monitoring.grafana_image: Invalid value: \"-\""))
				Expect(err.Error()).To(ContainSubstring("Docker image is invalid"))
			})
		})

		Context("invalid desired workers", func() {

			It("should return error", func() {
				rayCluster.Spec.DesiredWorkers = []string{"desired_workers-"}

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.DesiredWorkers: Invalid value: \"name\""))
				Expect(err.Error()).To(ContainSubstring("Must be DNS compliant name"))
			})
		})

		Context("invalid docker secret for jupyterhub", func() {

			It("should return error", func() {
				rayCluster.Spec.JupyterHub.DockerCredsSecret = "secret-"

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.jupyterhub: Invalid value: \"name\""))
				Expect(err.Error()).To(ContainSubstring("Must be DNS compliant name"))
			})
		})

		Context("invalid docker secret for monitoring", func() {

			It("should return error", func() {
				rayCluster.Spec.Monitoring.DockerCredsSecret = "secret-"

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.monitoring: Invalid value: \"name\""))
				Expect(err.Error()).To(ContainSubstring("Must be DNS compliant name"))
			})
		})
	})
}
