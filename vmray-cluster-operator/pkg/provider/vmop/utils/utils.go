// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/cloudinit"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	HeadNodeSecretSuffix   = "-hsecret"
	WorkerNodeSecretSuffix = "-wsecret"
	TokenSubResource       = "token"

	apiGroup            = "vmray.broadcom.com"
	rayClusterResources = "vmrayclusters"
	updateVerb          = "update"
	getVerb             = "get"

	kindRole = "Role"
	kindSa   = "ServiceAccount"
)

var (
	TokenExpirationRequest int64 = 60 * 60 * 30 * 6 // 6 months
)

// CreateCloudInitSecret checks if secret exists, otherwise creates it with necessary cloud init configuration.
func CreateCloudInitSecret(ctx context.Context,
	kubeclient client.Client,
	req provider.VmDeploymentRequest) (*corev1.Secret, bool, error) {

	secretName := req.VmName + WorkerNodeSecretSuffix
	if req.HeadNode {
		secretName = req.VmName + HeadNodeSecretSuffix
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
	// (Note: service account name & namespace are same as that of ray cluster CRD)
	token, err := fetchServiceAccountToken(ctx, kubeclient, req.Namespace, req.ClusterName)
	if err != nil {
		return nil, false, err
	}

	// If secret was not found, then create the secret.
	cloudInitSecret, err := cloudinit.CreateCloudConfigSecret(req.Namespace,
		req.ClusterName,
		secretName,
		token,
		req.NodeConfigSpec.VMUser,
		req.NodeConfigSpec.VMPasswordSaltHash,
		req.HeadNode)
	if err != nil {
		return nil, false, err
	}

	// create the secret.
	if err = kubeclient.Create(ctx, cloudInitSecret); err != nil {
		return nil, false, err
	}

	return cloudInitSecret, false, nil
}

func DeleteCloudInitSecret(ctx context.Context,
	kubeclient client.Client,
	req provider.VmDeploymentRequest) error {

	// 1. Check if the secret exists.
	secretName := req.VmName + WorkerNodeSecretSuffix
	if req.HeadNode {
		secretName = req.VmName + HeadNodeSecretSuffix
	}

	secretObjectkey := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      secretName,
	}

	// Check if secret exists.
	secret := &corev1.Secret{}
	if err := kubeclient.Get(ctx, secretObjectkey, secret); err != nil {
		// If err was NotFound then secret is already deleted, return without failure.
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return err
	}
	return kubeclient.Delete(ctx, secret)
}

func getVmRayClusterMutationRole(namespace, name string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name, // same as cluster name.
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{apiGroup},
				Resources: []string{rayClusterResources},
				Verbs:     []string{getVerb, updateVerb},
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

	// TODO : Add logging that sa, role & rolebinding was successfully created for given ray cluster.
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

	// TODO : Add logging that token was successfully created for given ray cluster.
	return tokenReq.Status.Token, err
}
