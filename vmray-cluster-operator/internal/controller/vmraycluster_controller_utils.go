// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"math/rand"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/internal/controller/lcm"
	vmprovider "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	alphanumeric = "abcdefghijlkmnopqrstuvwxyz0123456789"
)

func (r *VMRayClusterReconciler) reconcileHeadNode(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) error {
	setupLog.Info("Reconciling head node.")

	// Step 1: Get Node config required by head node.
	nodeConfig, err := r.getNodeConfig(ctx, instance.ObjectMeta.Namespace, instance.Spec.HeadNode.NodeConfigName)
	if err != nil {
		setupLog.Error(err, "Failure to get head node config", "name", instance.Spec.HeadNode.NodeConfigName)
		return err
	}

	nounce := instance.ObjectMeta.Labels[HeadNodeNounceLabel]
	req := lcm.NodeLcmRequest{
		Namespace:        instance.ObjectMeta.Namespace,
		Clustername:      instance.ObjectMeta.Name,
		Nounce:           nounce,
		Name:             vmprovider.GetHeadNodeName(instance.ObjectMeta.Name, nounce),
		DockerImage:      instance.Spec.Image,
		ApiServer:        instance.Spec.ApiServer,
		HeadNodeConfig:   instance.Spec.HeadNode,
		WorkerNodeConfig: instance.Spec.WorkerNode,
		NodeConfigSpec:   nodeConfig.Spec,
		NodeStatus:       &instance.Status.HeadNodeStatus,
		HeadNodeStatus:   nil,
	}

	// Step 2: leverage node lifecycle manager to process headnode state.
	err = r.nlcm.ProcessNodeVmState(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

func (r *VMRayClusterReconciler) reconcileWorkerNodes(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) error {
	setupLog.Info("Reconciling worker nodes.")

	// Step 0:
	// 0.1 : Initialize if current workers status map is empty.
	if instance.Status.CurrentWorkers == nil {
		instance.Status.CurrentWorkers = make(map[string]vmrayv1alpha1.VMRayNodeStatus)
	}

	// Step 1: Get Node config required to bring up worker nodes.
	nodeConfig, err := r.getNodeConfig(ctx, instance.Namespace, instance.Spec.WorkerNode.NodeConfigName)
	if err != nil {
		setupLog.Error(err, "Failure to fetch worker node config", "name", instance.Spec.WorkerNode.NodeConfigName)
		return err
	}

	// Step 2: Delete current worker nodes which are not mentioned in desired spec anymore.
	err = r.deleteWorkerNodes(ctx, instance, false)
	if err != nil {
		setupLog.Error(err, "Failed to delete nonessential worker nodes", "VMRayCluster", instance.Name)
		return err
	}

	// Step 3: Figure out list of new set of workers that needs to be added.
	// Leverage node lifecycle manager to process those VMs state.
	return r.reconcileDesiredWorkers(ctx, instance, nodeConfig)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (r *VMRayClusterReconciler) deleteWorkerNodes(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster, all bool) error {
	nodesToDelete := []string{}
	for name := range instance.Status.CurrentWorkers {
		if all || !contains(instance.Spec.DesiredWorkers, name) {
			nodesToDelete = append(nodesToDelete, name)
		}
	}
	return r.deleteNodes(ctx, nodesToDelete, instance)
}

func (r *VMRayClusterReconciler) deleteNodes(ctx context.Context, nodesToDelete []string, instance *vmrayv1alpha1.VMRayCluster) error {
	for _, name := range nodesToDelete {
		err := r.provider.Delete(ctx, instance.Namespace, name)
		if err != nil {
			return err
		}
		delete(instance.Status.CurrentWorkers, name)
		setupLog.Info("[DeleteWorkerNodes] Successfully deleted ray worker VM", "vm", name)
	}
	return nil
}

func (r *VMRayClusterReconciler) reconcileDesiredWorkers(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster, nodeConfig *vmrayv1alpha1.VMRayNodeConfig) error {

	currentWorkerNodes := []string{}
	for name := range instance.Status.CurrentWorkers {
		currentWorkerNodes = append(currentWorkerNodes, name)
	}

	for _, name := range instance.Spec.DesiredWorkers {
		// Check if worker is already present in current workers status map,
		// if so use those status objects during reconciliation, otherwise create
		// new status objects and assign them back.
		status := vmrayv1alpha1.VMRayNodeStatus{}
		if contains(currentWorkerNodes, name) {
			status = instance.Status.CurrentWorkers[name]
		}

		nounce := instance.ObjectMeta.Labels[HeadNodeNounceLabel]
		req := lcm.NodeLcmRequest{
			Namespace:      instance.ObjectMeta.Namespace,
			Clustername:    instance.ObjectMeta.Name,
			Nounce:         nounce,
			Name:           name,
			DockerImage:    instance.Spec.Image,
			HeadNodeConfig: instance.Spec.HeadNode,
			ApiServer:      instance.Spec.ApiServer,
			NodeConfigSpec: nodeConfig.Spec,
			NodeStatus:     &status,
			HeadNodeStatus: &instance.Status.HeadNodeStatus,
		}

		err = r.nlcm.ProcessNodeVmState(ctx, req)

		// reassign the status before checking for any errors.
		instance.Status.CurrentWorkers[name] = status
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *VMRayClusterReconciler) getNodeConfig(ctx context.Context, namespace, nodeConfigName string) (*vmrayv1alpha1.VMRayNodeConfig, error) {
	nodeConfig := &vmrayv1alpha1.VMRayNodeConfig{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      nodeConfigName,
	}
	// Get nodeconfig k8s custom resource hosting needed VM information.
	if err = r.Get(ctx, key, nodeConfig); err != nil {
		return nil, err
	}
	return nodeConfig, nil
}

func addErrorCondition(err error, instance *vmrayv1alpha1.VMRayCluster, Type, Reason string) {
	instance.Status.Conditions = append(instance.Status.Conditions, metav1.Condition{
		Type:               Type,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Reason:             Reason,
		Message:            err.Error(),
	})
}

func (r *VMRayClusterReconciler) updateStatus(ctx context.Context, re reconcileEnvelope) (ctrl.Result, error) {
	name := re.CurrentClusterState.ObjectMeta.Name
	status := re.CurrentClusterState.Status
	setupLog.Info("Update Ray cluster CR status", "name", name, "status", status)

	patch := client.MergeFrom(re.OriginalClusterState)
	err := r.Client.Status().Patch(ctx, re.CurrentClusterState, patch)
	if err != nil {
		setupLog.Error(err, "Error when updating status", "cluster name", name, "RayCluster", re.CurrentClusterState)
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, nil
}

func (r *VMRayClusterReconciler) getVMRayCluster(ctx context.Context, namespacedName types.NamespacedName, instance *vmrayv1alpha1.VMRayCluster) error {
	if err = r.Get(ctx, namespacedName, instance); err != nil {
		//Ignore not found errors
		if errors.IsNotFound(err) {
			setupLog.Info("Read request instance not found error!", "name", namespacedName)
		} else {
			setupLog.Error(err, "Read request instance error!")
		}
	}
	return err
}

// createRandomNounce generates a random alpha-numeric string of given size.
func createRandomNounce(n int) string {
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		buf[i] = alphanumeric[rand.Intn(len(alphanumeric))]
	}
	return string(buf)
}
