// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DockerUsernameKey = "username"
	DockerPasswordKey = "password"
	DockerRegistryKey = "registry"
	dockerLoginCmd    = "docker login %s --username %s --password %s"

	DockerKeyMissingErrorMsg = "Failure to authenticate with registry: secret `%s` is missing `%s` key"
)

type errorMissingKey struct {
	key        string
	secretName string
}

func newErrorMissingKey(secretName, key string) error {
	return &errorMissingKey{
		key:        key,
		secretName: secretName,
	}
}

func (e *errorMissingKey) Error() string {
	return fmt.Sprintf(DockerKeyMissingErrorMsg, e.secretName, e.key)
}

func GetDockerLoginCmd(ctx context.Context, kubeclient client.Client, namespace, secretName string) (string, error) {

	if len(secretName) == 0 {
		return "", nil
	}

	secretObjkey := client.ObjectKey{
		Namespace: namespace,
		Name:      secretName,
	}

	// Check if secret exists.
	var secret corev1.Secret
	var err error
	if err = kubeclient.Get(ctx, secretObjkey, &secret); err != nil {
		return "", err
	}

	// Get username from secret.
	username, ok := secret.Data[DockerUsernameKey]
	if !ok {
		return "", newErrorMissingKey(secretName, DockerUsernameKey)
	}

	// Get password from secret.
	password, ok := secret.Data[DockerPasswordKey]
	if !ok {
		return "", newErrorMissingKey(secretName, DockerPasswordKey)
	}

	// Get registry name from secret, (optional).
	var registry []byte
	registry, ok = secret.Data[DockerRegistryKey]
	if !ok {
		registry = []byte{}
	}

	// Generate docker login cmd.
	return fmt.Sprintf(dockerLoginCmd,
		string(registry),
		string(username),
		string(password)), nil
}
