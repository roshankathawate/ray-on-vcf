package tls_test

import (
	"context"
	"encoding/base64"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	vmrayv1alpha1 "github.com/vmware/ray-on-vcf/vmray-cluster-operator/api/v1alpha1"
	"github.com/vmware/ray-on-vcf/vmray-cluster-operator/pkg/provider"
	"github.com/vmware/ray-on-vcf/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
	"github.com/vmware/ray-on-vcf/vmray-cluster-operator/pkg/provider/vmop/tls"
	vmoputils "github.com/vmware/ray-on-vcf/vmray-cluster-operator/pkg/provider/vmop/utils"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func tlsTests() {
	var ns = "namespace-vmray"

	Describe("Create TLS configs, secrets, certs and keys", func() {

		clusterName := "cluster-name-1"
		vmName := "vm-name"
		req := provider.VmDeploymentRequest{
			Namespace:      ns,
			ClusterName:    clusterName,
			VmName:         vmName,
			HeadNodeStatus: nil,
			EnableTLS:      true,
			NodeConfig: vmrayv1alpha1.CommonNodeConfig{
				VMUser:             "rayvm-user",
				VMPasswordSaltHash: "rayvm-salthash",
			},
		}

		Context("Validate successful submission of root ca certs and key for ray cluster", func() {
			It("Verify `CreateVMRayClusterRootSecret` & `CreateCloudInitSecret` with TLS enabled", func() {

				k8sClient := suite.GetK8sClient()

				// Create the needed namespace.
				nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
				err := k8sClient.Create(context.Background(), nsSpec)
				Expect(err).ToNot(HaveOccurred())

				// Test Service account, role & rolebinding creation.
				err = vmoputils.CreateServiceAccountAndRole(context.Background(), k8sClient, ns, clusterName)
				Expect(err).ToNot(HaveOccurred())

				// Test Setup Root Ca for VMRayCluster
				err = tls.CreateVMRayClusterRootSecret(context.Background(), k8sClient, ns, clusterName)
				Expect(err).ToNot(HaveOccurred())

				secretObjectkey := client.ObjectKey{
					Namespace: ns,
					Name:      clusterName + tls.RootCaSecretSuffix,
				}
				// Check TLS secret is created.
				tlsSecret := &corev1.Secret{}
				err = k8sClient.Get(context.Background(), secretObjectkey, tlsSecret)
				Expect(err).ToNot(HaveOccurred())
				Expect(tlsSecret.ObjectMeta.Name).To(Equal(clusterName + tls.RootCaSecretSuffix))

				// Create the Head Node secret.
				secret, alreadyExists, err := vmoputils.CreateCloudInitSecret(context.Background(), k8sClient, req)
				Expect(err).ToNot(HaveOccurred())
				Expect(alreadyExists).To(BeFalse())
				Expect(secret.ObjectMeta.Name).To(Equal(clusterName + vmoputils.HeadNodeSecretSuffix))

				// Decode base64 cloud init.
				decodedcloudinit, err := base64.StdEncoding.DecodeString(string(secret.Data[cloudinit.CloudInitConfigUserDataKey]))
				Expect(err).ToNot(HaveOccurred())

				// Validate CaCrt was set in CloudConfig.
				caCrt := string(secret.Data[cloudinit.Ca_cert_file])
				Expect(caCrt).NotTo(BeNil())

				// Validate CaKey was set in CloudConfig
				caKey := string(secret.Data[cloudinit.Ca_key_file])
				Expect(caKey).NotTo(BeNil())

				// Extract TLS related environment variables injected into cloud init config.
				rayDockerRunCommandString := ""
				for _, sentence := range strings.Split(string(decodedcloudinit), "\n") {
					if strings.Contains(sentence, "docker run") {
						rayDockerRunCommandString = sentence
						break
					}
				}
				Expect(rayDockerRunCommandString).To(ContainSubstring("RAY_USE_TLS=1"))
				Expect(rayDockerRunCommandString).To(ContainSubstring("RAY_TLS_CA_CERT=/home/ray/ca.crt"))
				Expect(rayDockerRunCommandString).To(ContainSubstring("RAY_TLS_SERVER_KEY=/home/ray/tls.key"))
				Expect(rayDockerRunCommandString).To(ContainSubstring("RAY_TLS_SERVER_CERT=/home/ray/tls.crt"))

				// Validate secret reuse.
				_, alreadyExists, err = vmoputils.CreateCloudInitSecret(context.Background(), k8sClient, req)
				Expect(err).ToNot(HaveOccurred())
				Expect(alreadyExists).To(BeTrue())

				Expect(k8sClient).NotTo(BeNil())
			})

			It("Verify `CreateVMRayClusterRootSecret` & `CreateCloudInitSecret` with TLS disabled", func() {
				clusterName := "cluster-name-2"
				vmName := "vm-name-2"
				req := provider.VmDeploymentRequest{
					Namespace:      ns,
					ClusterName:    clusterName,
					VmName:         vmName,
					HeadNodeStatus: nil,
					EnableTLS:      false,
					NodeConfig: vmrayv1alpha1.CommonNodeConfig{
						VMUser:             "rayvm-user",
						VMPasswordSaltHash: "rayvm-salthash",
					},
				}

				k8sClient := suite.GetK8sClient()

				// Test Service account, role & rolebinding creation.
				err := vmoputils.CreateServiceAccountAndRole(context.Background(), k8sClient, ns, clusterName)
				Expect(err).ToNot(HaveOccurred())

				// Test Setup Root Ca for VMRayCluster
				err = tls.CreateVMRayClusterRootSecret(context.Background(), k8sClient, ns, clusterName)
				Expect(err).ToNot(HaveOccurred())

				secretObjectkey := client.ObjectKey{
					Namespace: ns,
					Name:      clusterName + tls.RootCaSecretSuffix,
				}
				// Check TLS secret is created.
				tlsSecret := &corev1.Secret{}
				err = k8sClient.Get(context.Background(), secretObjectkey, tlsSecret)
				Expect(err).ToNot(HaveOccurred())
				Expect(tlsSecret.ObjectMeta.Name).To(Equal(clusterName + tls.RootCaSecretSuffix))

				// Create the Head Node secret.
				secret, alreadyExists, err := vmoputils.CreateCloudInitSecret(context.Background(), k8sClient, req)
				Expect(err).ToNot(HaveOccurred())
				Expect(alreadyExists).To(BeFalse())
				Expect(secret.ObjectMeta.Name).To(Equal(clusterName + vmoputils.HeadNodeSecretSuffix))

				// Decode base64 cloud init.
				decodedcloudinit, err := base64.StdEncoding.DecodeString(string(secret.Data[cloudinit.CloudInitConfigUserDataKey]))
				Expect(err).ToNot(HaveOccurred())

				// Validate CaCrt was set in CloudConfig.
				caCrt := string(secret.Data[cloudinit.Ca_cert_file])
				Expect(caCrt).NotTo(BeNil())

				// Validate CaKey was set in CloudConfig
				caKey := string(secret.Data[cloudinit.Ca_key_file])
				Expect(caKey).NotTo(BeNil())

				// Extract TLS related environment variables injected into cloud init config.
				rayDockerRunCommandString := ""
				for _, sentence := range strings.Split(string(decodedcloudinit), "\n") {
					if strings.Contains(sentence, "docker run") {
						rayDockerRunCommandString = sentence
						break
					}
				}
				Expect(rayDockerRunCommandString).To(ContainSubstring("RAY_USE_TLS=0"))
				Expect(rayDockerRunCommandString).To(ContainSubstring("RAY_TLS_CA_CERT=/home/ray/ca.crt"))
				Expect(rayDockerRunCommandString).To(ContainSubstring("RAY_TLS_SERVER_KEY=/home/ray/tls.key"))
				Expect(rayDockerRunCommandString).To(ContainSubstring("RAY_TLS_SERVER_CERT=/home/ray/tls.crt"))

			})

			It("Verify `DeleteCloudInitSecret` & `DeleteServiceAccountAndRole`", func() {
				k8sClient := suite.GetK8sClient()

				// Validate deletion of secret & auxiliary k8s resources.
				err := vmoputils.DeleteAllCloudInitSecret(context.Background(), k8sClient, ns, clusterName)
				Expect(err).ToNot(HaveOccurred())

				err = vmoputils.DeleteServiceAccountAndRole(context.Background(), k8sClient, ns, clusterName)
				Expect(err).ToNot(HaveOccurred())

				// Second attempt for deletion of secret & k8s resource should
				// be pass a through, and shouldnt encounter any failures.
				err = vmoputils.DeleteAllCloudInitSecret(context.Background(), k8sClient, ns, clusterName)
				Expect(err).ToNot(HaveOccurred())

				err = vmoputils.DeleteServiceAccountAndRole(context.Background(), k8sClient, ns, clusterName)
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})
}
