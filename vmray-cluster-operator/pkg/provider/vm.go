// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
)

type VmDeploymentRequest struct {
	Namespace      string
	ClusterName    string
	VmUser         string
	VmPasswordHash string
	VmName         string
	DockerImage    string

	NodeConfigSpec vmrayv1alpha1.VMRayNodeConfigSpec

	// If nil then it's the head node. If non nil then it's the worker node.
	// The worker node requires some properties of head node like IP to be cloudinit
	HeadNodeStatus *vmrayv1alpha1.VMRayNodeStatus
}

type VmProvider interface {
	Deploy(context.Context, VmDeploymentRequest) error
	Delete(context.Context, string, string) error
	FetchVmStatus(context.Context, string, string) (*vmrayv1alpha1.VMRayNodeStatus, error)
}
