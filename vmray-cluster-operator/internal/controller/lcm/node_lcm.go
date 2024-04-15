// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
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

func (nlcm *NodeLifecycleManager) ProcessNodeVmState(ctx context.Context,
	namespace, clustername, name string,
	nodeConfig *vmrayv1alpha1.VMRayNodeConfig,
	status *vmrayv1alpha1.VMRayNodeStatus) error {

	log := ctrl.LoggerFrom(ctx)
	switch status.VmStatus {
	case "":
		// Case where node is not created and request just came in so its status is not set.
		// TODO: Provide docker image.
		deploymentRequest := provider.VmDeploymentRequest{
			Namespace:   namespace,
			ClusterName: clustername,
			VmName:      name,
			NodeConfigSpec: vmrayv1alpha1.VMRayNodeConfigSpec{
				VMUser:             nodeConfig.Spec.VMUser,
				VMPasswordSaltHash: nodeConfig.Spec.VMPasswordSaltHash,
				VMImage:            nodeConfig.Spec.VMImage,
				VMClass:            nodeConfig.Spec.VMClass,
				StorageClass:       nodeConfig.Spec.StorageClass,
			},
		}
		if err := nlcm.pvdr.Deploy(ctx, deploymentRequest); err != nil {
			log.Error(err, "Got error when deploying ray head node")
			return err
		}

		// Update node vm status to initialized.
		status.VmStatus = vmrayv1alpha1.INITIALIZED

		log.Info("Deployed node and set its status to INITIALIZED", "VM", name)

	case vmrayv1alpha1.INITIALIZED:
		// Check if node is created, validate if node IP is assigned.
		newStatus, err := nlcm.pvdr.FetchVmStatus(ctx, namespace, name)
		if err != nil {
			log.Error(err, "Got error when fetching VM status in INITIALIZED node state")
			return err
		}
		// Update status as per node's VM crd.
		status.Name = newStatus.Name
		status.Ip = newStatus.Ip
		status.Conditions = newStatus.Conditions

		if status.Ip == "" {
			// VM is still not up, keep the current state.
			return nil
		}
		// If IP is assigned move the VM status to RUNNING state.
		status.VmStatus = vmrayv1alpha1.RUNNING
		status.RayStatus = vmrayv1alpha1.RAY_INITIALIZED

		log.Info("IP assignment is successful and set node status to RUNNING", "VM", name)

	case vmrayv1alpha1.RUNNING:
		// Validate if node IP is still available.
		newStatus, err := nlcm.pvdr.FetchVmStatus(ctx, namespace, name)
		if err == nil && newStatus.Ip != "" {
			// Update conditions change.
			status.Conditions = newStatus.Conditions
			return nil
		}

		log.Error(err, "Detected failure moving node to Failed state", "VM", name)
		status.VmStatus = vmrayv1alpha1.FAIL
		status.RayStatus = vmrayv1alpha1.RAY_FAIL

		if err == nil && newStatus.Ip != "" {
			err = fmt.Errorf("Primary IPv4 not found for %s Node", name)
		}
		return err

	case vmrayv1alpha1.FAIL:
		// TODO: Do we want to perform any operation here ?
	default:
	}
	return nil
}
