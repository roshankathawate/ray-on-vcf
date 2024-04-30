// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
)

type VmDeploymentRequest struct {
	ClusterName    string
	Commands       []string
	DockerImage    string
	Namespace      string
	VmName         string
	ApiServer      vmrayv1alpha1.ApiServerInfo
	NodeConfigSpec vmrayv1alpha1.VMRayNodeConfigSpec

	// Leveraged only during ray head VM deployment.
	IdleTimeoutMinutes uint
	MaxWorkers         uint
	MinWorkers         uint

	// Leveraged only during ray worker VM deployment. If nil then
	// it's the head node, if non-nil then it is for worker node.
	// The worker node requires some properties of head node like
	// IP to be in cloudinit.
	HeadNodeStatus *vmrayv1alpha1.VMRayNodeStatus
}

type VmProvider interface {
	Deploy(context.Context, VmDeploymentRequest) error
	Delete(context.Context, string, string) error
	FetchVmStatus(context.Context, string, string) (*vmrayv1alpha1.VMRayNodeStatus, error)
	DeleteAuxiliaryResources(context.Context, string, string) error
}
