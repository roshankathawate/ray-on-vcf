// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"encoding/base64"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
	vmoputils "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/utils"

	jwtv4 "github.com/golang-jwt/jwt/v4"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

type claimsK8s struct {
	ExpiresAt int64 `json:"exp,omitempty"`
	IssuedAt  int64 `json:"iat,omitempty"`
}

func (cn claimsK8s) Valid() error {
	return nil
}

func cloudInitSecretCreationTests() {
	var ns = "namespace-vmray"

	Describe("Create service account, role & binding", func() {

		clusterName := "cluster-name-1"
		vmName := "vm-name"
		req := provider.VmDeploymentRequest{
			Namespace:      ns,
			ClusterName:    clusterName,
			VmName:         vmName,
			HeadNodeStatus: nil,
			NodeConfigSpec: vmrayv1alpha1.VMRayNodeConfigSpec{
				VMUser:             "vm-username",
				VMPasswordSaltHash: "salt-hash",
			},
		}

		Context("Validate successful submission of required svc account for ray cluster", func() {
			It("Verify `CreateServiceAccountAndRole` & `CreateCloudInitSecret` function logics", func() {

				k8sClient := suite.GetK8sClient()

				// Create the needed namespace.
				nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
				err := k8sClient.Create(context.Background(), nsSpec)
				Expect(err).To(BeNil())

				// Test Service account, role & rolebinding creation.
				err = vmoputils.CreateServiceAccountAndRole(context.Background(), k8sClient, ns, clusterName)
				Expect(err).To(BeNil())

				// Create the secret.
				secret, alreadyExists, err := vmoputils.CreateCloudInitSecret(context.Background(), k8sClient, req)
				Expect(err).To(BeNil())
				Expect(alreadyExists).To(Equal(false))
				Expect(secret.ObjectMeta.Name).To(Equal(clusterName + vmoputils.HeadNodeSecretSuffix))

				// Decode base64 cloud init.
				decodedcloudinit, err := base64.StdEncoding.DecodeString(string(secret.Data[cloudinit.CloudInitConfigUserDataKey]))
				Expect(err).To(BeNil())

				// Extract service account token injected into cloud init config.
				svcAccStr := ""
				for _, sentence := range strings.Split(string(decodedcloudinit), "\n") {
					if strings.Contains(sentence, "SVC_ACCOUNT_TOKEN") {
						svcAccStr = sentence
						break
					}
				}

				svcAccStr = strings.SplitAfter(svcAccStr, "=")[1]
				svcAccStr = strings.SplitAfter(svcAccStr, " ")[0]
				svcAccStr = strings.TrimSpace(svcAccStr)
				svcAccStr = strings.Trim(svcAccStr, "\"")
				Expect(svcAccStr).NotTo(Equal(""))

				// Extract Jwt token using regex.
				re := regexp.MustCompile(`[A-Za-z0-9-_]*\.[A-Za-z0-9-_]*\.[A-Za-z0-9-_]*$`)
				matches := re.FindStringSubmatch(svcAccStr)
				Expect(matches).NotTo(BeNil())

				// Validate token was created with specificed number fof Exp seconds
				claims := &claimsK8s{}
				_, _, err = jwtv4.NewParser().ParseUnverified(matches[0], claims)
				Expect(err).To(BeNil())
				Expect(claims.ExpiresAt - claims.IssuedAt).To(Equal(vmoputils.TokenExpirationRequest))

				// Validate secret reuse.
				_, alreadyExists, err = vmoputils.CreateCloudInitSecret(context.Background(), k8sClient, req)
				Expect(err).To(BeNil())
				Expect(alreadyExists).To(Equal(true))

				Expect(k8sClient).NotTo(BeNil())
			})

			It("Verify `DeleteCloudInitSecret` & `DeleteServiceAccountAndRole` function logics", func() {
				k8sClient := suite.GetK8sClient()

				// Validate deletion of secret & auxiliary k8s resources.
				err := vmoputils.DeleteAllCloudInitSecret(context.Background(), k8sClient, ns, clusterName)
				Expect(err).To(BeNil())

				err = vmoputils.DeleteServiceAccountAndRole(context.Background(), k8sClient, ns, clusterName)
				Expect(err).To(BeNil())

				// Second attempt for deletion of secret & k8s resource should
				// be pass a through, and shouldnt encounter any failures.
				err = vmoputils.DeleteAllCloudInitSecret(context.Background(), k8sClient, ns, clusterName)
				Expect(err).To(BeNil())

				err = vmoputils.DeleteServiceAccountAndRole(context.Background(), k8sClient, ns, clusterName)
				Expect(err).To(BeNil())
			})
		})

		Context("Verifying updation to VMRayCluster CRD using service account token", func() {
			It("Create service account, role & role binding and verfiy VMRayCluster CRD can be updated", func() {

				k8sClient := suite.GetK8sClient()
				ctx := context.Background()

				// Create service account, role & rolebinding.
				err := vmoputils.CreateServiceAccountAndRole(ctx, k8sClient, ns, clusterName)
				Expect(err).To(BeNil())

				// Create token.
				sa := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterName,
						Namespace: ns,
					},
				}

				tokenReq := &authenticationv1.TokenRequest{
					Spec: authenticationv1.TokenRequestSpec{
						ExpirationSeconds: &vmoputils.TokenExpirationRequest,
					},
				}
				err = k8sClient.SubResource(vmoputils.TokenSubResource).Create(ctx, sa, tokenReq)
				Expect(err).To(BeNil())

				// 1. Create third party client using bearer token
				scheme := runtime.NewScheme()
				err = vmrayv1alpha1.AddToScheme(scheme)
				Expect(err).NotTo(HaveOccurred())

				extclient, err := client.New(&rest.Config{
					Host:        suite.GetApiServer(),
					BearerToken: tokenReq.Status.Token,
					TLSClientConfig: rest.TLSClientConfig{
						Insecure: true,
					},
				}, client.Options{Scheme: scheme})
				Expect(err).To(BeNil())
				Expect(extclient).NotTo(BeNil())

				// 2. Check if external client can create the CRD, this should fail with Forbidden error.
				head_node := vmrayv1alpha1.HeadNodeConfig{
					NodeConfigName: "head_node",
				}
				worker_node := vmrayv1alpha1.WorkerNodeConfig{
					NodeConfigName: "worker_node",
					MinWorkers:     0,
					MaxWorkers:     1,
				}
				jupyterhub := &vmrayv1alpha1.JupyterHubConfig{
					Image:             "quay.io/jupyterhub/jupyterhub",
					DockerCredsSecret: "secret",
				}
				monitoring := &vmrayv1alpha1.MonitoringConfig{
					PrometheusImage:   "prom/prometheus",
					GrafanaImage:      "grafana/grafana-oss",
					DockerCredsSecret: "secret",
				}
				vmraycluster := &vmrayv1alpha1.VMRayCluster{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ns,
						Name:      clusterName,
					},
					Spec: vmrayv1alpha1.VMRayClusterSpec{
						Image:      "rayproject/ray:2.5.0",
						HeadNode:   head_node,
						WorkerNode: worker_node,
						JupyterHub: jupyterhub,
						Monitoring: monitoring,
					},
				}
				err = extclient.Create(ctx, vmraycluster)
				Expect(k8serrors.IsForbidden(err)).To(Equal(true))

				// 3. Create CRD using controller k8s client, and update it using externnal client.
				err = k8sClient.Create(ctx, vmraycluster)
				Expect(err).To(BeNil())

				// Try to update it using external client.
				err = extclient.Update(ctx, vmraycluster)
				Expect(err).To(BeNil())

				// Try and fetch it using external client.
				key := client.ObjectKey{
					Namespace: ns,
					Name:      clusterName,
				}

				getvmray := &vmrayv1alpha1.VMRayCluster{}
				err = extclient.Get(ctx, key, getvmray)
				Expect(err).To(BeNil())
				Expect(getvmray.Spec.Monitoring.GrafanaImage).To(Equal("grafana/grafana-oss"))
			})
		})

	})
}
