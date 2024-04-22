// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit_test

import (
	"encoding/base64"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	format "github.com/onsi/gomega/format"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
)

func templatingTests() {
	var vmDeploymentRequest provider.VmDeploymentRequest
	var cloudConfig cloudinit.CloudConfig

	// If the output is too large, go truncates it to 4000 characters.
	// Inorder to get over that default behaviour, we set format.MaxLength to 0
	format.MaxLength = 0
	dockerImage := "harbor-repo.vmware.com/fudata/ray-on-vsphere:py38"

	Describe("Cloudinit templating modules", func() {
		BeforeEach(func() {
			vmDeploymentRequest = provider.VmDeploymentRequest{
				Namespace:      "namespace-head",
				ClusterName:    "clustername",
				HeadNodeStatus: nil,
				DockerImage:    dockerImage,
				NodeConfigSpec: vmrayv1alpha1.VMRayNodeConfigSpec{
					VMUser:             "rayvm-user",
					VMPasswordSaltHash: "rayvm-salthash",
				},
			}

			cloudConfig.VmDeploymentRequest = vmDeploymentRequest
			cloudConfig.SecretName = "headvm-cloud-config-secret"
			cloudConfig.SvcAccToken = "token-val1"
		})
		Context("Validate cloud config secret creation for the head node", func() {
			It("Create cloud config for head node", func() {
				secret, err := cloudinit.CreateCloudInitConfigSecret(cloudConfig)
				Expect(err).To(BeNil())
				b64Str := secret.StringData[cloudinit.CloudInitConfigUserDataKey]

				data, err := base64.StdEncoding.DecodeString(b64Str)
				dataStr := string(data[:])

				Expect(err).To(BeNil())
				Expect(secret).To(ContainSubstring("headvm-cloud-config-secret"))
				Expect(dataStr).To(ContainSubstring(dockerImage))
			})
		})

		Context("Validate cloud config secret creation for the worker node", func() {
			It("Create cloud config for worker node", func() {

				vmDeploymentRequest.NodeConfigSpec.VMUser = "rayvm-user2"
				vmDeploymentRequest.Namespace = "namespace-worker"
				vmDeploymentRequest.HeadNodeStatus = &v1alpha1.VMRayNodeStatus{
					Ip: "12.12.12.12",
				}

				cloudConfig.SecretName = "headvm-cloud-config-secret"
				cloudConfig.SvcAccToken = "token-val2"
				cloudConfig.VmDeploymentRequest = vmDeploymentRequest

				secret, err := cloudinit.CreateCloudInitConfigSecret(cloudConfig)

				Expect(err).To(BeNil())
				Expect(secret.ObjectMeta.Namespace).To(Equal("namespace-worker"))

				b64Str := secret.StringData[cloudinit.CloudInitConfigUserDataKey]

				data, err := base64.StdEncoding.DecodeString(b64Str)

				Expect(err).To(BeNil())

				dataStr := string(data[:])

				Expect(dataStr).To(ContainSubstring("12.12.12.12"))
				Expect(dataStr).To(ContainSubstring("ray start --address=$RAY_HEAD_IP:6379 --object-manager-port=8076"))
			})
		})
	})
}
