// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package lcm

import (
	"context"
	"errors"
	"fmt"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrorInvalidNodestatus = errors.New("lcm detected invalid node status")
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
	Namespace   string
	Clustername string
	Nounce      string
	Name        string
	NodeType    string
	DockerImage string
	ApiServer   vmrayv1alpha1.ApiServerInfo
	EnableTLS   bool

	// Head & common node configs.
	HeadNodeConfig   vmrayv1alpha1.HeadNodeConfig
	WorkerNodeConfig vmrayv1alpha1.WorkerNodeConfig
	NodeConfig       vmrayv1alpha1.CommonNodeConfig
	DockerConfig     vmrayv1alpha1.DockerRegistryConfig

	// Dymamically tracked states.
	NodeStatus      *vmrayv1alpha1.VMRayNodeStatus
	HeadNodeStatus  *vmrayv1alpha1.VMRayNodeStatus
	VMServiceStatus *vmrayv1alpha1.VMServiceStatus

	// This specifics what form was leveraged
	// to submit ray cluster request.
	RayClusterRequestor provider.RayClusterRequestor
}

func (nlcm *NodeLifecycleManager) ProcessNodeVmState(ctx context.Context, req NodeLcmRequest) error {

	log := ctrl.LoggerFrom(ctx)
	switch req.NodeStatus.VmStatus {
	case vmrayv1alpha1.EMPTY:
		// Case where node is not created and request just came in so its status is not set.
		deploymentRequest := provider.VmDeploymentRequest{
			Namespace:           req.Namespace,
			ClusterName:         req.Clustername,
			Nounce:              req.Nounce,
			VmName:              req.Name,
			NodeType:            req.NodeType,
			DockerImage:         req.DockerImage,
			HeadNodeStatus:      req.HeadNodeStatus,
			ApiServer:           req.ApiServer,
			HeadNodeConfig:      req.HeadNodeConfig,
			NodeConfig:          req.NodeConfig,
			WorkerNodeConfig:    req.WorkerNodeConfig,
			EnableTLS:           req.EnableTLS,
			RayClusterRequestor: req.RayClusterRequestor,
			DockerConfig:        req.DockerConfig,
		}

		// Get Fetch or Create VM service construct before deploying head vm.
		if req.HeadNodeStatus == nil {
			if ip, err := nlcm.pvdr.DeployVmService(ctx, deploymentRequest); err != nil {
				log.Error(err, "Got error when deploying/fetching ray vm service")
				return err
			} else if ip == "" {
				log.Info("VM service IP is not ready, try in the next reconcile loop")
				return nil
			} else {
				// Set vm service IP in cluster status and head vm deployment.
				deploymentRequest.VmService = ip
				req.VMServiceStatus.Ip = ip
			}
		}

		// Deploy VM.
		if err := nlcm.pvdr.Deploy(ctx, deploymentRequest); err != nil {
			if client.IgnoreAlreadyExists(err) != nil {
				log.Error(err, "Got error when deploying ray head/worker node")
				req.NodeStatus.VmStatus = vmrayv1alpha1.FAIL
				return err
			}
			log.Error(err, "Ignoring VM CRD already exists error")
		}

		// Update node vm status to initialized.
		log.Info("Deployed node and set its status to INITIALIZED", "VM", req.Name)
		req.NodeStatus.VmStatus = vmrayv1alpha1.INITIALIZED

	case vmrayv1alpha1.INITIALIZED:
		// Check if node is created, validate if node IP is assigned.
		newStatus, err := nlcm.pvdr.FetchVmStatus(ctx, req.Namespace, req.Name)
		if err != nil {
			log.Error(err, "Got error when fetching VM status in INITIALIZED node state")
			req.NodeStatus.VmStatus = vmrayv1alpha1.FAIL
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

		if err == nil && newStatus.Ip == "" {
			err = fmt.Errorf("primary IPv4 not found for %s Node", req.Name)
		}

		log.Error(err, "Detected failure moving node to Failed state", "VM", req.Name)
		req.NodeStatus.VmStatus = vmrayv1alpha1.FAIL
		req.NodeStatus.RayStatus = vmrayv1alpha1.RAY_FAIL

		return err

	case vmrayv1alpha1.FAIL:
		// Try to fetch VM CRD, to validate if its available.
		_, err := nlcm.pvdr.FetchVmStatus(ctx, req.Namespace, req.Name)
		if err == nil {
			// Set status to `INITIALIZED` mode.
			log.Info("VM CRD detected, Status changed from FAIL to INITIALIZED", "VM", req.Name)
			req.NodeStatus.VmStatus = vmrayv1alpha1.INITIALIZED
			return nil
		}

		// If VM CRD is not available, we need to redeploy the VM.
		if client.IgnoreNotFound(err) == nil {
			log.Info("VM CRD is not detected, Status changed from FAIL to `empty string` (i.e. creation request mode)", "VM", req.Name)
			req.NodeStatus.VmStatus = vmrayv1alpha1.EMPTY
		}

		log.Info("Failing to fetch VM status, node marked as failure", "VM", req.Name)
	default:
		log.Error(ErrorInvalidNodestatus, "Invalid node status detected", "VM", req.Name, "Status", req.NodeStatus.VmStatus)
		return ErrorInvalidNodestatus
	}
	return nil
}
