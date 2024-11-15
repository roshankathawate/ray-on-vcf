// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
)

const (
	headsuffix                = "-h"
	RayClusterRequestorLabel  = "vmray.io/created-by"
	RayClusterRequestorRayCLI = "ray-cli"
)

type RayClusterRequestor int

const (
	RayCLI RayClusterRequestor = iota + 1
	K8S
)

func (r RayClusterRequestor) IsRayCli() bool {
	return r == RayCLI
}

type VmDeploymentRequest struct {
	ClusterName string
	DockerImage string
	Namespace   string
	Nounce      string
	VmName      string
	NodeType    string
	ApiServer   vmrayv1alpha1.ApiServerInfo
	EnableTLS   bool

	// Head & common node configs.
	HeadNodeConfig   vmrayv1alpha1.HeadNodeConfig
	WorkerNodeConfig vmrayv1alpha1.WorkerNodeConfig
	NodeConfig       vmrayv1alpha1.CommonNodeConfig
	DockerConfig     vmrayv1alpha1.DockerRegistryConfig

	// Leveraged only during ray worker VM deployment. If nil then
	// it's the head node, if non-nil then it is for worker node.
	// The worker node requires some properties of head node like
	// IP to be in cloudinit.
	HeadNodeStatus *vmrayv1alpha1.VMRayNodeStatus

	// This specifics what form was leveraged
	// to submit ray cluster request.
	RayClusterRequestor RayClusterRequestor

	// VmService represents ingress IP associated with the said VM.
	VmService string
}

type VmProvider interface {
	Deploy(context.Context, VmDeploymentRequest) error
	DeployVmService(context.Context, VmDeploymentRequest) (string, error)
	Delete(context.Context, string, string) error
	FetchVmStatus(context.Context, string, string) (*vmrayv1alpha1.VMRayNodeStatus, error)
	DeleteAuxiliaryResources(context.Context, string, string) error
}

func GetHeadNodeName(clustername, nounce string) string {
	res := clustername + headsuffix
	if len(nounce) > 0 {
		res = res + "-" + nounce
	}
	return res
}
