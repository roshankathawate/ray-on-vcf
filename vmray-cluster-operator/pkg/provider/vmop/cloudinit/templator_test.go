// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit_test

import (
	"encoding/base64"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
)

func templatingTests() {
	Describe("Cloudinit templating modules", func() {

		Context("Validate cloud config secret creation", func() {
			It("Create cloud config for head node", func() {
				secret, err := cloudinit.CreateCloudConfigSecret("namespace-head", "clusternam", "headvm-cloud-config-secret", "token-val1", "rayvm-user", "rayvm-salthash", true)
				Expect(err).To(BeNil())

				Expect(secret.ObjectMeta.Namespace).To(Equal("namespace-head"))

				b64Str := secret.StringData[cloudinit.CloudConfigUserDataKey]

				dataBytes, err := base64.StdEncoding.DecodeString(b64Str)
				Expect(err).To(BeNil())

				data := string(dataBytes)
				Expect(len(strings.SplitAfter(data, "\n"))).To(Equal(86))
				Expect(data).To(ContainSubstring("     service_account_token: token-val1"))
				Expect(data).To(ContainSubstring("    passwd: \"rayvm-salthash\""))
			})

			It("Create cloud config for worker node", func() {
				secret, err := cloudinit.CreateCloudConfigSecret("namespace-worker", "clustername", "worker-cloud-config-secret", "token-val2", "rayvm-user2", "rayvm-salthash", false)
				Expect(err).To(BeNil())
				Expect(secret.ObjectMeta.Namespace).To(Equal("namespace-worker"))

				b64Str := secret.StringData[cloudinit.CloudConfigUserDataKey]

				dataBytes, err := base64.StdEncoding.DecodeString(b64Str)
				Expect(err).To(BeNil())

				data := string(dataBytes)
				Expect(len(strings.SplitAfter(data, "\n"))).To(Equal(17))
				Expect(data).To(Not(ContainSubstring("     service_account_token: token-val2")))
				Expect(data).To(ContainSubstring("su rayvm-user2 -c '/home/rayvm-user2/ray-env/bin/ray start --address=$RAY_HEAD_IP:6379 --object-manager-port=8076 > ~/ray_worker_startup.log'"))
			})
		})
	})
}
