// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package lcm

import (
	"context"
	"fmt"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	ctrl "sigs.k8s.io/controller-runtime"
)

type NodeLifecycleManager struct {
	pvdr provider.VmProvider
}

func NewNodeLifecycleManager(pvdr provider.VmProvider) *NodeLifecycleManager {
	return &NodeLifecycleManager{
		pvdr: pvdr,
	}
}

type NodeLcmRequest struct {
	Namespace      string
	Clustername    string
	Name           string
	DockerImage    string
	Commands       []string
	ApiServer      vmrayv1alpha1.ApiServerInfo
	NodeConfigSpec vmrayv1alpha1.VMRayNodeConfigSpec

	// Leveraged only during Head VM deployment.
	IdleTimeoutMinutes uint
	MaxWorkers         uint
	MinWorkers         uint

	// Dymamically tracked states.
	NodeStatus     *vmrayv1alpha1.VMRayNodeStatus
	HeadNodeStatus *vmrayv1alpha1.VMRayNodeStatus
}

func (nlcm *NodeLifecycleManager) ProcessNodeVmState(ctx context.Context, req NodeLcmRequest) error {

	log := ctrl.LoggerFrom(ctx)
	switch req.NodeStatus.VmStatus {
	case "":
		// Case where node is not created and request just came in so its status is not set.
		deploymentRequest := provider.VmDeploymentRequest{
			Namespace:          req.Namespace,
			ClusterName:        req.Clustername,
			VmName:             req.Name,
			DockerImage:        req.DockerImage,
			NodeConfigSpec:     req.NodeConfigSpec,
			HeadNodeStatus:     req.HeadNodeStatus,
			ApiServer:          req.ApiServer,
			IdleTimeoutMinutes: req.IdleTimeoutMinutes,
			MaxWorkers:         req.MaxWorkers,
			MinWorkers:         req.MinWorkers,
			Commands:           req.Commands,
		}
		if err := nlcm.pvdr.Deploy(ctx, deploymentRequest); err != nil {
			log.Error(err, "Got error when deploying ray head/worker node")
			req.NodeStatus.VmStatus = vmrayv1alpha1.FAIL
			return err
		}

		// Update node vm status to initialized.
		log.Info("Deployed node and set its status to INITIALIZED", "VM", req.Name)
		req.NodeStatus.VmStatus = vmrayv1alpha1.INITIALIZED

	case vmrayv1alpha1.INITIALIZED:
		// Check if node is created, validate if node IP is assigned.
		newStatus, err := nlcm.pvdr.FetchVmStatus(ctx, req.Namespace, req.Name)
		if err != nil {
			log.Error(err, "Got error when fetching VM status in INITIALIZED node state")
			return err
		}
		// Update status as per node's VM crd.
		req.NodeStatus.Ip = newStatus.Ip
		req.NodeStatus.Conditions = newStatus.Conditions

		if req.NodeStatus.Ip == "" {
			// VM is still not up, keep the current state.
			return nil
		}
		// If IP is assigned move the VM status to RUNNING state.
		req.NodeStatus.VmStatus = vmrayv1alpha1.RUNNING
		req.NodeStatus.RayStatus = vmrayv1alpha1.RAY_INITIALIZED

		log.Info("IP assignment is successful and set node status to RUNNING", "VM", req.Name)

	case vmrayv1alpha1.RUNNING:
		// Validate if node IP is still available.
		newStatus, err := nlcm.pvdr.FetchVmStatus(ctx, req.Namespace, req.Name)
		if err == nil && newStatus.Ip != "" {

			// TODO: Check ray status on node.
			req.NodeStatus.RayStatus = vmrayv1alpha1.RAY_RUNNING

			req.NodeStatus.Ip = newStatus.Ip
			req.NodeStatus.Conditions = newStatus.Conditions
			log.Info("VM & Ray process are in RUNNING state.", "VM", req.Name)
			return nil
		}

		log.Error(err, "Detected failure moving node to Failed state", "VM", req.Name)
		req.NodeStatus.VmStatus = vmrayv1alpha1.FAIL
		req.NodeStatus.RayStatus = vmrayv1alpha1.RAY_FAIL

		if err == nil && newStatus.Ip != "" {
			err = fmt.Errorf("Primary IPv4 not found for %s Node", req.Name)
		}
		return err

	case vmrayv1alpha1.FAIL:
		// TODO: Do we want to perform any operation here ?
	default:
	}
	return nil
}
