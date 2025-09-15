// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vmware/ray-on-vcf/vmray-cluster-operator/api/v1alpha1"
	. "github.com/vmware/ray-on-vcf/vmray-cluster-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func vmRayClusterUnitTests() {
	var (
		rayCluster VMRayCluster
	)
	Describe("VMRayCluster validating webhook", func() {

		port := uint(6379)

		BeforeEach(func() {
			head_node := HeadNodeConfig{
				Port:     &port,
				NodeType: "worker_1",
			}
			rayCluster = VMRayCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "valid-name",
				},
				Spec: VMRayClusterSpec{
					Image:    "rayproject/ray:2.5.0",
					HeadNode: head_node,
					NodeConfig: CommonNodeConfig{
						VMImage:      "vm-image",
						StorageClass: "storage-class",
						NodeTypes: map[string]NodeType{
							"worker_1": {
								VMClass:    "vm-class",
								MinWorkers: 3,
								MaxWorkers: 5,
							},
						},
					},
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

		Context("invalid Min/Max workers", func() {

			It("should return error", func() {
				nt := rayCluster.Spec.NodeConfig.NodeTypes["worker_1"]
				nt.MinWorkers = 4
				nt.MaxWorkers = 1
				rayCluster.Spec.NodeConfig.NodeTypes["worker_1"] = nt

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.common_node_config.node_types.worker_1: Invalid value: \"min_workers/max_workers\""))
				Expect(err.Error()).To(ContainSubstring("Min workers cannot be more than Max workers, min_workers: 4, max_workers: 1"))
			})
		})

		Context("invalid total min workers & invalid max_worker in available node type", func() {
			It("should return error", func() {
				rayCluster.Spec.NodeConfig.MaxWorkers = 5

				nt := rayCluster.Spec.NodeConfig.NodeTypes["worker_1"]
				nt.MinWorkers = 3
				nt.MaxWorkers = 5
				rayCluster.Spec.NodeConfig.NodeTypes["worker_1"] = nt

				nt = v1alpha1.NodeType{}
				nt.MinWorkers = 3
				nt.MaxWorkers = 4
				rayCluster.Spec.NodeConfig.NodeTypes["worker_2"] = nt

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Expected spec.common_node_config.max_workers is: 5, but total desired min worker for all available node type is: 6"))
			})
		})

		Context("invalid max_worker in available node type", func() {
			It("should return error", func() {
				rayCluster.Spec.NodeConfig.MaxWorkers = 5

				nt := rayCluster.Spec.NodeConfig.NodeTypes["worker_1"]
				nt.MinWorkers = 2
				nt.MaxWorkers = 5
				rayCluster.Spec.NodeConfig.NodeTypes["worker_1"] = nt

				nt = v1alpha1.NodeType{}
				nt.MinWorkers = 3
				nt.MaxWorkers = 6
				rayCluster.Spec.NodeConfig.NodeTypes["worker_2"] = nt

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Available node type max worker count shoudn't be more than cluster max_worker count, node_type: worker_2, spec.common_node_config.max_workers: 5"))
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

		Context("invalid desired workers", func() {

			It("should return error", func() {
				rayCluster.Spec.AutoscalerDesiredWorkers = map[string]string{
					"desired_workers-": "node-type-1",
				}

				err := suite.GetK8sClient().Create(context.TODO(), &rayCluster)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("spec.autoscaler_desired_workers: Invalid value: \"name\""))
				Expect(err.Error()).To(ContainSubstring("Must be DNS compliant name"))
			})
		})
	})
}
