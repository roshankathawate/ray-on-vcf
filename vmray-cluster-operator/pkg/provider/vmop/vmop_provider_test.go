// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package vmop_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"

	vmprovider "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop"
	tls_utils "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/tls"
	vmoputils "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/utils"
)

const (
	namespace = "namespace-test"
)

func VmOpProviderTests() {

	Describe("Validate functions exposed to support VM lifecycle via vmoperator CRD", func() {

		Context("Validate Deploy, Delete and FetchVmStatus", func() {
			dockerImage := "Docker-image:ray"
			ctx := context.Background()

			It("Create vm provider and validate CR creation in local env", func() {

				clustername := "cluster-name"
				vmname := "vm-name-1"
				k8sClient := suite.GetK8sClient()
				provider := vmop.NewVmOperatorProvider(k8sClient)

				// Create the needed namespace.
				nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
				err := k8sClient.Create(ctx, nsSpec)
				Expect(err).ToNot(HaveOccurred())

				err = tls_utils.CreateVMRayClusterRootSecret(ctx, k8sClient, namespace, clustername)
				Expect(err).ToNot(HaveOccurred())

				// Validiate Deploy function.
				vmDeploymentRequest := vmprovider.VmDeploymentRequest{
					Namespace:      namespace,
					ClusterName:    clustername,
					VmName:         vmname,
					DockerImage:    dockerImage,
					EnableTLS:      true,
					NodeType:       "ray_head",
					HeadNodeStatus: nil,
					// Head & common node configs.
					HeadNodeConfig: vmrayv1alpha1.HeadNodeConfig{},
					NodeConfig: vmrayv1alpha1.CommonNodeConfig{
						VMUser:             "rayvm-user",
						VMPasswordSaltHash: "rayvm-salthash",
						VMImage:            "vmi-00001",
						StorageClass:       "storage-default",
						MaxWorkers:         3,
						NodeTypes: map[string]vmrayv1alpha1.NodeType{
							"worker_1": {
								VMClass:    "best-effort-xsmall",
								MaxWorkers: 5,
								MinWorkers: 3,
								Resources: vmrayv1alpha1.NodeResource{
									CPU:    4,
									Memory: 1000,
								},
							},
							"ray_head": {
								VMClass: "best-effort-xlarge",
							},
						},
					},
				}

				// 1. Deploy VMs and aux k8s resources.
				err = provider.Deploy(ctx, vmDeploymentRequest)
				Expect(err).ToNot(HaveOccurred())

				vmNamespaceName := types.NamespacedName{
					Name:      vmname,
					Namespace: namespace,
				}
				vminstance := &vmopv1.VirtualMachine{}
				err = k8sClient.Get(ctx, vmNamespaceName, vminstance)
				Expect(err).ToNot(HaveOccurred())
				Expect(vminstance.Spec.ClassName).To(Equal("best-effort-xlarge"))

				// 2. Fetch VM status.
				_, err = provider.FetchVmStatus(ctx, namespace, vmname)
				Expect(err).ToNot(HaveOccurred())

				// 3. Validate aux resources exists and later delete them.
				exists, err := checkIfSecretExists(k8sClient, namespace, clustername+vmoputils.HeadNodeSecretSuffix)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())

				err = provider.DeleteAuxiliaryResources(ctx, namespace, clustername)
				Expect(err).ToNot(HaveOccurred())

				exists, err = checkIfSecretExists(k8sClient, namespace, clustername+vmoputils.HeadNodeSecretSuffix)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())

				// 4. Delete VM and its VM Service.
				err = provider.Delete(ctx, namespace, vmname)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
}

func checkIfSecretExists(kubeclient client.Client, namespace, name string) (bool, error) {
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	// Check if secret exists.
	secret := &corev1.Secret{}
	if err := kubeclient.Get(context.Background(), key, secret); err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}
