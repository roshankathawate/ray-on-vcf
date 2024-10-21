// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"

	vmprovider "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/tls"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

const (
	HeadNodeSecretSuffix   = "-hsecret"
	WorkerNodeSecretSuffix = "-wsecret"
	sshKeySecretSuffix     = "-ssh-key"
	TokenSubResource       = "token"
	SshPrivateKey          = "ssh-pvt-key"

	rayApiGroup         = "vmray.broadcom.com"
	vmopApiGroup        = "vmoperator.vmware.com"
	rayClusterResources = "vmrayclusters"
	vmServiceResources  = "virtualmachineservices"
	patchVerb           = "patch"
	getVerb             = "get"

	kindRole = "Role"
	kindSa   = "ServiceAccount"

	error_tmpl_pvt_key = "Failure to read ssh key: secret %s:%s doesn't contain `%s` key"
)

var (
	TokenExpirationRequest int64 = 60 * 60 * 30 * 6 // 6 months
)

// CreateCloudInitSecret checks if secret exists, otherwise creates it
// with necessary cloud init configuration.
func CreateCloudInitSecret(ctx context.Context,
	kubeclient client.Client,
	req vmprovider.VmDeploymentRequest) (*corev1.Secret, bool, error) {

	var cloudConfig cloudinit.CloudConfig
	var err error

	nodeSecretName := req.ClusterName + WorkerNodeSecretSuffix
	if req.HeadNodeStatus == nil {
		nodeSecretName = req.ClusterName + HeadNodeSecretSuffix
		var err error
		// Create private ssh key to be set for all nodes in cluster secret.
		cloudConfig.SshPvtKey, err = getOrCreatePrivateKeySecret(ctx,
			kubeclient, req.Namespace, req.ClusterName)
		if err != nil {
			return nil, false, err
		}
	} else {
		// Read ssh private key from cluster's ssh keys store secret.
		cloudConfig.SshPvtKey, err = readPrivateKeyForCluster(ctx,
			kubeclient, req.Namespace, req.ClusterName)
		if err != nil {
			return nil, false, err
		}
	}

	nodeSecretObjectkey := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      nodeSecretName,
	}

	// Check if secret exists.
	var validSecret corev1.Secret
	if err := kubeclient.Get(ctx, nodeSecretObjectkey, &validSecret); err == nil {
		return &validSecret, true, nil
	} else if client.IgnoreNotFound(err) != nil {
		return nil, false, err
	}

	// Create long lived token using service account
	// (Note: service account name & namespace are same
	// as that of ray cluster CRD)
	token, err := fetchServiceAccountToken(ctx, kubeclient, req.Namespace, req.ClusterName)
	if err != nil {
		return nil, false, err
	}

	cloudConfig.VmDeploymentRequest = req
	cloudConfig.SvcAccToken = token
	cloudConfig.SecretName = nodeSecretName

	ca_key, ca_crt, err := tls.ReadCaCrtAndCaKeyFromSecret(
		ctx, kubeclient, req.Namespace, req.ClusterName+tls.TLSSecretSuffix)
	cloudConfig.CaCrt = ca_crt
	cloudConfig.CaKey = ca_key
	if err != nil {
		return nil, false, err
	}

	// Get docker login cmd.
	dockerLoginCmd, err := GetDockerLoginCmd(ctx, kubeclient, req.Namespace, req.DockerConfig.AuthSecretName)
	if err != nil {
		return nil, false, err
	}
	cloudConfig.DockerLoginCmd = dockerLoginCmd

	// If secret was not found, then create the secret.
	cloudInitSecret, err := cloudinit.CreateCloudInitConfigSecret(cloudConfig)
	if err != nil {
		return nil, false, err
	}

	// create the secret.
	if err = kubeclient.Create(ctx, cloudInitSecret); err != nil {
		return nil, false, err
	}

	return cloudInitSecret, false, nil
}

func DeleteAllCloudInitSecret(ctx context.Context,
	kubeclient client.Client, namespace, clusterName string) error {

	// Delete ssh keys store secret.
	err := deleteSecret(ctx, kubeclient, namespace, GetSshKeysSecretName(clusterName))
	if err != nil {
		return err
	}

	// Delete worker config secret.
	err = deleteSecret(ctx, kubeclient, namespace, clusterName+WorkerNodeSecretSuffix)
	if err != nil {
		return err
	}

	// Delete TLS config secret.
	err = deleteSecret(ctx, kubeclient, namespace, clusterName+tls.TLSSecretSuffix)
	if err != nil {
		return err
	}

	// Delete head config secret.
	return deleteSecret(ctx, kubeclient, namespace, clusterName+HeadNodeSecretSuffix)
}

func deleteSecret(ctx context.Context,
	kubeclient client.Client, namespace, name string) error {

	secretObjectkey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	// Check if secret exists.
	secret := &corev1.Secret{}
	if err := kubeclient.Get(ctx, secretObjectkey, secret); err != nil {
		// If err was NotFound then secret is already deleted, return without failure.
		return client.IgnoreNotFound(err)
	}
	return kubeclient.Delete(ctx, secret)
}

func GetSshKeysSecretName(name string) string {
	return name + sshKeySecretSuffix
}

func readPrivateKeyForCluster(ctx context.Context,
	kubeclient client.Client, namespace, name string) (string, error) {

	secretObjectkey := client.ObjectKey{
		Namespace: namespace,
		Name:      GetSshKeysSecretName(name),
	}

	// Check if secret exists.
	secret := &corev1.Secret{}
	if err := kubeclient.Get(ctx, secretObjectkey, secret); err != nil {
		return "", err
	}

	pvt_key, ok := secret.Data[SshPrivateKey]
	if !ok {
		return "", fmt.Errorf(error_tmpl_pvt_key, namespace, name, SshPrivateKey)
	}

	return string(pvt_key), nil
}

func getVmRayClusterMutationRole(namespace, name string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name, // same as cluster name.
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{rayApiGroup},
				Resources: []string{rayClusterResources},
				Verbs:     []string{getVerb, patchVerb},
			},
			{
				APIGroups: []string{vmopApiGroup},
				Resources: []string{vmServiceResources},
				Verbs:     []string{getVerb},
			},
		},
	}
}

func getVmRayClusterMutationRoleBinding(namespace, name string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name, // same as vmray cluster name.
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     kindRole,
			Name:     name, // same as vmray cluster name.
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      kindSa,
				Name:      name,      // same as vmray cluster name.
				Namespace: namespace, // Name of the Namespace
			},
		},
	}
}

func CreateServiceAccountAndRole(ctx context.Context, kubeclient client.Client, namespace, name string) error {

	// Common cluster key for all k8s resource types.
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	// Check if service account exist otherwise create for specific cluster.
	// to be leverage by autoscaler in head node.
	sa := &corev1.ServiceAccount{}
	if err := kubeclient.Get(ctx, key, sa); err != nil {
		// If error is `Not Found`, move to create service account.
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		sa = &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		err := kubeclient.Create(ctx, sa)
		if err != nil {
			return err
		}
	}

	// Create role defining update verb on VMRaycluster CRD.
	role := getVmRayClusterMutationRole(namespace, name)

	// Define role binding to link service account and role.
	roleBinding := getVmRayClusterMutationRoleBinding(namespace, name)

	// Check if role exist otherwise create for specific cluster.
	if err := kubeclient.Get(ctx, key, role); err != nil {
		// If error is `Not Found`, move to create role
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		err := kubeclient.Create(ctx, role)
		if err != nil {
			return err
		}
	}

	// Check if role binding exist otherwise create it for specific cluster.
	if err := kubeclient.Get(ctx, key, roleBinding); err != nil {
		// If error is `Not Found`, move to create role
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		err := kubeclient.Create(ctx, roleBinding)
		if err != nil {
			return err
		}
	}

	// TODO: Add logging that sa, role & rolebinding was successfully created for given ray cluster.
	return nil
}

// DeleteServiceAccountAndRole performs deletion of auxiliary k8s resources in opposite order as of their creation.
func DeleteServiceAccountAndRole(ctx context.Context, kubeclient client.Client, namespace, name string) error {

	// Common cluster key for all k8s resource types.
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	// 1. Delete rolebinding
	rb := &rbacv1.RoleBinding{}
	if err := kubeclient.Get(ctx, key, rb); err != nil {
		// If error is `Not Found`, move to deleting role.
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	} else {
		// If no error is encountered, then role binding exists and delete it.
		if err := kubeclient.Delete(ctx, rb); err != nil {
			return err
		}
	}

	// 2. Delete role
	role := &rbacv1.Role{}
	if err := kubeclient.Get(ctx, key, role); err != nil {
		// If error is `Not Found`, move to deleting service account
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	} else {
		// If no error is encountered, then role exists so delete it.
		if err := kubeclient.Delete(ctx, role); err != nil {
			return err
		}
	}

	// 3. Delete service account
	sa := &corev1.ServiceAccount{}
	if err := kubeclient.Get(ctx, key, sa); err != nil {
		// If error is `Not Found`, then return nil.
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		return nil
	}
	return kubeclient.Delete(ctx, sa)
}

func fetchServiceAccountToken(ctx context.Context, kubeclient client.Client, namespace, name string) (string, error) {

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	tokenReq := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &TokenExpirationRequest,
		},
	}
	err := kubeclient.SubResource(TokenSubResource).Create(ctx, sa, tokenReq)
	if err != nil {
		return "", err
	}

	// TODO: Add logging that token was successfully created for given ray cluster.
	return tokenReq.Status.Token, err
}

// Logic to generate a private RSA key.
// source: https://earthly.dev/blog/encrypting-data-with-ssh-keys-and-golang/
func marshalRSAPrivate(priv *rsa.PrivateKey) string {
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	}))
}

func createPrivateSshKey() string {
	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return ""
	}

	return marshalRSAPrivate(key)
}

func getOrCreatePrivateKeySecret(ctx context.Context,
	kubeclient client.Client, namespace, name string) (string, error) {

	// Check if secret exists and extract private ssh key
	if pvt_key, err := readPrivateKeyForCluster(ctx, kubeclient, namespace, name); err == nil {
		return pvt_key, nil
	} else if client.IgnoreNotFound(err) != nil {
		return "", err
	}

	// create the secret if it doesn't exists, which happens
	// when a request comes from user using CRD submission
	// and not from autoscaler CLI up.
	pvt_key := createPrivateSshKey()

	dataMap := map[string]string{
		SshPrivateKey: pvt_key,
	}

	sshSecret := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetSshKeysSecretName(name),
			Namespace: namespace,
		},
		StringData: dataMap,
	}

	if err := kubeclient.Create(ctx, sshSecret); err != nil {
		return "", err
	}
	return pvt_key, nil
}
