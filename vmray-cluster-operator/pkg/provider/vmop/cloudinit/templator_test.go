// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit_test

import (
	"encoding/base64"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	format "github.com/onsi/gomega/format"
	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
)

const (
	certContent      = "-----BEGIN CERTIFICATE-----\nca-cert-value\n-----END CERTIFICATE-----"
	secretName       = "headvm-cloud-config-secret"
	keyContent       = "-----BEGIN RSA PRIVATE KEY-----\nca-key-value\n-----END RSA PRIVATE KEY-----"
	caCertString     = "RAY_TLS_CA_CERT=/home/ray/ca.crt"
	tlsKeyString     = "RAY_TLS_SERVER_KEY=/home/ray/tls.key"
	tlsCertString    = "RAY_TLS_SERVER_CERT=/home/ray/tls.crt"
	genCertString    = "sh /home/ray/gencert.sh"
	enableTLSString  = "RAY_USE_TLS=1"
	disableTLSString = "RAY_USE_TLS=0"
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
				Namespace:      "namespace",
				ClusterName:    "clustername",
				HeadNodeStatus: nil,
				DockerImage:    dockerImage,
				EnableTLS:      true,
				NodeConfig: vmrayv1alpha1.CommonNodeConfig{
					VMUser:             "rayvm-user",
					VMPasswordSaltHash: "rayvm-salthash",
				},
			}

			cloudConfig.VmDeploymentRequest = vmDeploymentRequest
			cloudConfig.SecretName = secretName
			cloudConfig.SvcAccToken = "token-val1"
			cloudConfig.SshPvtKey = "-----BEGIN RSA PRIVATE KEY-----\nssh-private-key-value\n-----END RSA PRIVATE KEY-----"
			cloudConfig.CaCrt = certContent
			cloudConfig.CaKey = keyContent
		})
		Context("Validate cloud config secret creation for the head node with TLS enabled", func() {
			It("Create cloud config for head node", func() {

				init_cmds := []string{"echo \"init cmds - 1\"", "echo \"init cmds - 2\""}
				cloudConfig.VmDeploymentRequest.NodeConfig.InitializationCommands = init_cmds
				cloudConfig.VmDeploymentRequest.NodeConfig.SetupCommands = []string{"echo \"common cmds - 1\""}
				cloudConfig.VmDeploymentRequest.HeadNodeConfig.SetupCommands = []string{"echo \"head cmds - 1\""}
				cloudConfig.VmDeploymentRequest.WorkerNodeConfig.SetupCommands = []string{"echo \"worker cmds - 1\""}

				secret, err := cloudinit.CreateCloudInitConfigSecret(cloudConfig)
				Expect(err).ToNot(HaveOccurred())
				b64Str := secret.StringData[cloudinit.CloudInitConfigUserDataKey]

				data, err := base64.StdEncoding.DecodeString(b64Str)
				Expect(err).ToNot(HaveOccurred())

				dataStr := string(data[:])
				Expect(secret).To(ContainSubstring(secretName))
				Expect(dataStr).To(ContainSubstring(dockerImage))
				Expect(dataStr).To(ContainSubstring(enableTLSString))
				Expect(dataStr).To(ContainSubstring(caCertString))
				Expect(dataStr).To(ContainSubstring(tlsKeyString))
				Expect(dataStr).To(ContainSubstring(tlsCertString))
				Expect(dataStr).To(ContainSubstring(genCertString))
				Expect(dataStr).To(ContainSubstring("- su rayvm-user -c 'echo \"init cmds - 1\"'"))
				Expect(dataStr).To(ContainSubstring("- su rayvm-user -c 'echo \"init cmds - 2\"'"))
				Expect(dataStr).To(ContainSubstring("sh /home/ray/gencert.sh;ray stop;ray start --head --port=6379 " +
					"--block --autoscaling-config=/home/ray/ray_bootstrap_config.yaml --dashboard-host=0.0.0.0"))
			})
		})
		Context("Validate cloud config secret creation for the head node with TLS disabled", func() {
			It("Create cloud config for head node", func() {
				vmDeploymentRequest.EnableTLS = false
				vmDeploymentRequest.VmService = "12.12.12.12"
				cloudConfig.VmDeploymentRequest = vmDeploymentRequest

				secret, err := cloudinit.CreateCloudInitConfigSecret(cloudConfig)
				Expect(err).ToNot(HaveOccurred())
				b64Str := secret.StringData[cloudinit.CloudInitConfigUserDataKey]

				data, err := base64.StdEncoding.DecodeString(b64Str)
				dataStr := string(data[:])

				Expect(err).ToNot(HaveOccurred())
				Expect(secret).To(ContainSubstring(secretName))
				Expect(dataStr).To(ContainSubstring(dockerImage))
				Expect(dataStr).To(ContainSubstring(disableTLSString))
				Expect(dataStr).To(ContainSubstring(caCertString))
				Expect(dataStr).To(ContainSubstring(tlsKeyString))
				Expect(dataStr).To(ContainSubstring(tlsCertString))
				Expect(dataStr).To(ContainSubstring(genCertString))
				Expect(dataStr).To(ContainSubstring("RAY_VMSERVICE_IP=12.12.12.12"))
			})
		})

		Context("Validate cloud config secret creation for the worker node", func() {
			It("Create cloud config for worker node", func() {

				vmDeploymentRequest.NodeConfig.VMUser = "rayvm-user2"
				vmDeploymentRequest.Namespace = "namespace-worker"
				vmDeploymentRequest.HeadNodeStatus = &vmrayv1alpha1.VMRayNodeStatus{}

				cloudConfig.SecretName = secretName
				cloudConfig.SvcAccToken = "token-val2"
				cloudConfig.VmDeploymentRequest = vmDeploymentRequest
				cloudConfig.CaCrt = certContent
				cloudConfig.CaKey = keyContent

				secret, err := cloudinit.CreateCloudInitConfigSecret(cloudConfig)

				Expect(err).ToNot(HaveOccurred())
				Expect(secret.ObjectMeta.Namespace).To(Equal("namespace-worker"))

				b64Str := secret.StringData[cloudinit.CloudInitConfigUserDataKey]

				data, err := base64.StdEncoding.DecodeString(b64Str)

				Expect(err).ToNot(HaveOccurred())

				dataStr := string(data[:])

				Expect(dataStr).To(ContainSubstring("/home/rayvm-user2/.ssh/id_rsa_ray.pub"))
			})
		})
	})
}
