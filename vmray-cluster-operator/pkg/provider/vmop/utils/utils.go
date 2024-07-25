// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"fmt"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmprovider "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
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
	TokenSubResource       = "token"

	rayApiGroup         = "vmray.broadcom.com"
	vmopApiGroup        = "vmoperator.vmware.com"
	rayClusterResources = "vmrayclusters"
	vmServiceResources  = "virtualmachineservices"
	patchVerb           = "patch"
	getVerb             = "get"

	kindRole = "Role"
	kindSa   = "ServiceAccount"

	Protocol_TCP = "TCP"

	error_tmpl_pvt_key = "Failure to read private key: secret %s:%s doesn't contain `%s` key"
)

var (
	TokenExpirationRequest int64 = 60 * 60 * 30 * 6 // 6 months
)

// CreateCloudInitSecret checks if secret exists, otherwise creates it with necessary cloud init configuration.
func CreateCloudInitSecret(ctx context.Context,
	kubeclient client.Client,
	req vmprovider.VmDeploymentRequest) (*corev1.Secret, bool, error) {

	var cloudConfig cloudinit.CloudConfig

	secretName := req.ClusterName + WorkerNodeSecretSuffix
	if req.HeadNodeStatus == nil {
		secretName = req.ClusterName + HeadNodeSecretSuffix
		// Create private ssh key to be set for all node in head node's secret.
		cloudConfig.SshPvtKey = createSshPrivateKey()
	} else {
		headnode := vmprovider.GetHeadNodeName(req.ClusterName, req.Nounce)
		ingressIp, err := isVmServiceUp(ctx, kubeclient, req.Namespace, headnode)
		if err != nil {
			return nil, false, err
		}
		cloudConfig.HeadVmServiceIngressIp = ingressIp

		// Read ssh private key from head node's secret.
		headNodeSecretName := req.ClusterName + HeadNodeSecretSuffix
		cloudConfig.SshPvtKey, err = readPrivateKeyFromNode(ctx, kubeclient, req.Namespace, headNodeSecretName)
		if err != nil {
			return nil, false, err
		}
	}

	secretObjectkey := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      secretName,
	}

	// Check if secret exists.
	var validSecret corev1.Secret
	if err := kubeclient.Get(ctx, secretObjectkey, &validSecret); err == nil {
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
	cloudConfig.SecretName = secretName

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

	// Delete worker config secret.
	err := deleteSecret(ctx, kubeclient, namespace, clusterName+WorkerNodeSecretSuffix)
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

func readPrivateKeyFromNode(ctx context.Context,
	kubeclient client.Client, namespace, name string) (string, error) {

	secretObjectkey := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	// Check if secret exists.
	secret := &corev1.Secret{}
	if err := kubeclient.Get(ctx, secretObjectkey, secret); err != nil {
		return "", err
	}
	pvt_key, ok := secret.Data[cloudinit.SshPrivateKey]
	if !ok {
		return "", fmt.Errorf(error_tmpl_pvt_key, namespace, name, cloudinit.SshPrivateKey)
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

func CreateVMService(ctx context.Context, kubeclient client.Client, namespace, name string,
	ports map[string]int32, selector map[string]string) error {

	vmserviceport := []vmopv1.VirtualMachineServicePort{}
	for n, p := range ports {
		v := vmopv1.VirtualMachineServicePort{
			Name:       n,
			Protocol:   Protocol_TCP,
			Port:       p,
			TargetPort: p,
		}
		vmserviceport = append(vmserviceport, v)
	}

	vmservice := &vmopv1.VirtualMachineService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: vmopv1.VirtualMachineServiceSpec{
			Selector: selector,
			Ports:    vmserviceport,
			Type:     vmopv1.VirtualMachineServiceTypeLoadBalancer,
		},
	}

	// key for cluster's head node's VMService object.
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	// Check if VMService exist otherwise create for specific cluster head node.
	if err := kubeclient.Get(ctx, key, vmservice); err != nil {
		// If error is `Not Found`, move to create VM service.
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		err := kubeclient.Create(ctx, vmservice)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteVMService(ctx context.Context, kubeclient client.Client, namespace, name string) error {
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	// Check if vmservice exists.
	vmservice := &vmopv1.VirtualMachineService{}
	if err := kubeclient.Get(ctx, key, vmservice); err != nil {
		// If err was NotFound then vmservice is already deleted, return without failure.
		return client.IgnoreNotFound(err)
	}
	return kubeclient.Delete(ctx, vmservice)
}

func isVmServiceUp(ctx context.Context, kubeclient client.Client, namespace, name string) (string, error) {
	// key for cluster's ray node's VMService object.
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	// Check if VMService exists and fetch its service ingress IP.
	vmservice := &vmopv1.VirtualMachineService{}
	if err := kubeclient.Get(ctx, key, vmservice); err != nil {
		return "", err
	}

	ingress := vmservice.Status.LoadBalancer.Ingress
	if len(ingress) > 0 {
		return ingress[0].IP, nil
	}
	return "", errors.New("Head node VM service IP is not assigned")
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

func createSshPrivateKey() string {
	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return ""
	}
	return marshalRSAPrivate(key)
}
