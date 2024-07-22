// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"math/rand"
	"time"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/internal/controller/lcm"
	vmprovider "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/constants"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"slices"
)

const (
	alphanumeric = "abcdefghijlkmnopqrstuvwxyz0123456789"
)

func (r *VMRayClusterReconciler) reconcileHeadNode(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) error {
	r.Log.Info("Reconciling head node.")

	nounce := instance.ObjectMeta.Labels[HeadNodeNounceLabel]
	req := lcm.NodeLcmRequest{
		Namespace:      instance.ObjectMeta.Namespace,
		Clustername:    instance.ObjectMeta.Name,
		Nounce:         nounce,
		Name:           vmprovider.GetHeadNodeName(instance.ObjectMeta.Name, nounce),
		NodeType:       constants.DefaultHeadNodeType,
		DockerImage:    instance.Spec.Image,
		ApiServer:      instance.Spec.ApiServer,
		HeadNodeConfig: instance.Spec.HeadNode,
		NodeConfig:     instance.Spec.NodeConfig,
		NodeStatus:     &instance.Status.HeadNodeStatus,
		HeadNodeStatus: nil,
	}

	// Step 2: leverage node lifecycle manager to process headnode state.
	if err := r.nlcm.ProcessNodeVmState(ctx, req); err != nil {
		return err
	}

	return nil
}

func (r *VMRayClusterReconciler) reconcileWorkerNodes(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) error {
	r.Log.Info("Reconciling worker nodes.")

	// Step 0:
	// 0.1 : Initialize if current workers status map is empty.
	if instance.Status.CurrentWorkers == nil {
		instance.Status.CurrentWorkers = make(map[string]vmrayv1alpha1.VMRayNodeStatus)
	}

	// If desired autoscaler map is empty, initialise it.
	if instance.Spec.AutoscalerDesiredWorkers == nil {
		instance.Spec.AutoscalerDesiredWorkers = make(map[string]string)
	}

	// Step 1: Delete current worker nodes which are not mentioned in desired spec anymore.
	if err := r.deleteWorkerNodes(ctx, instance, false); err != nil {
		r.Log.Error(err, "Failed to delete nonessential worker nodes", "VMRayCluster", instance.ObjectMeta.Name)
		return err
	}

	// Step 2: Figure out list of new set of workers that needs to be added.
	// Leverage node lifecycle manager to process those VMs state.
	return r.reconcileDesiredWorkers(ctx, instance)
}

func (r *VMRayClusterReconciler) deleteWorkerNodes(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster, all bool) error {
	nodesToDelete := []string{}
	for name := range instance.Status.CurrentWorkers {
		if _, ok := instance.Spec.AutoscalerDesiredWorkers[name]; all || !ok {
			nodesToDelete = append(nodesToDelete, name)
		}
	}
	return r.deleteNodes(ctx, nodesToDelete, instance)
}

func (r *VMRayClusterReconciler) deleteNodes(ctx context.Context, nodesToDelete []string, instance *vmrayv1alpha1.VMRayCluster) error {
	for _, name := range nodesToDelete {
		err := r.provider.Delete(ctx, instance.ObjectMeta.Namespace, name)
		if err != nil {
			return err
		}
		delete(instance.Status.CurrentWorkers, name)
		r.Log.Info("[DeleteWorkerNodes] Successfully deleted ray worker VM", "vm", name)
	}
	return nil
}

func (r *VMRayClusterReconciler) reconcileDesiredWorkers(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) error {

	for name, nodeTypeName := range instance.Spec.AutoscalerDesiredWorkers {
		// Check if worker is already present in current workers status map,
		// if so use those status objects during reconciliation, otherwise create
		// new status objects and assign them back.
		status := vmrayv1alpha1.VMRayNodeStatus{}
		if s, ok := instance.Status.CurrentWorkers[name]; ok {
			status = s
		}

		nounce := instance.ObjectMeta.Labels[HeadNodeNounceLabel]
		req := lcm.NodeLcmRequest{
			Namespace:      instance.ObjectMeta.Namespace,
			Clustername:    instance.ObjectMeta.Name,
			Nounce:         nounce,
			Name:           name,
			NodeType:       nodeTypeName,
			DockerImage:    instance.Spec.Image,
			HeadNodeConfig: instance.Spec.HeadNode,
			NodeConfig:     instance.Spec.NodeConfig,
			ApiServer:      instance.Spec.ApiServer,
			NodeStatus:     &status,
			HeadNodeStatus: &instance.Status.HeadNodeStatus,
		}

		err := r.nlcm.ProcessNodeVmState(ctx, req)

		// reassign the status before checking for any errors.
		instance.Status.CurrentWorkers[name] = status
		if err != nil {
			return err
		}
	}
	return nil
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
	r.Log.Info("Update Ray cluster CR status", "name", name, "status", status)

	patch := client.MergeFrom(re.OriginalClusterState)
	err := r.Client.Status().Patch(ctx, re.CurrentClusterState, patch)
	if err != nil {
		r.Log.Error(err, "Error when updating status", "cluster name", name, "RayCluster", re.CurrentClusterState)
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, nil
}

func (r *VMRayClusterReconciler) fetchVMRayCluster(ctx context.Context, namespacedName types.NamespacedName, instance *vmrayv1alpha1.VMRayCluster) error {
	err := r.Client.Get(ctx, namespacedName, instance)
	if err != nil {
		//  Ignore not found errors.
		if k8serrors.IsNotFound(err) {
			r.Log.Error(err, "Read request instance not found", "name", namespacedName)
		} else {
			r.Log.Error(err, "Read request instance error")
		}
	}
	return err
}

func (r *VMRayClusterReconciler) ValidateAuxiliaryDependencies(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) (bool, error) {

	// Validate existence of VMI, storage class and
	// vm classs associated with-in namespace.

	invalidState := false

	// 1. Validate VM image
	vmi := vmopv1.VirtualMachineImage{}
	vmiNamespaceName := types.NamespacedName{
		Name:      instance.Spec.NodeConfig.VMImage,
		Namespace: instance.ObjectMeta.Namespace,
	}
	if err := r.Client.Get(ctx, vmiNamespaceName, &vmi); err != nil {
		if errors.IsNotFound(err) {
			addErrorCondition(err, instance, vmrayv1alpha1.NodeConfigInvalidVMI, vmrayv1alpha1.ResourceNotFoundReason)
		} else {
			r.Log.Error(err, "Failure when trying to fetch VM image.", "Namespace", vmiNamespaceName.Namespace, "Name", vmiNamespaceName.Name)
			return true, err
		}
		invalidState = true
	}

	// 2. Validate Storage class.
	sc := storagev1.StorageClass{}
	scNamespaceName := types.NamespacedName{
		Name:      instance.Spec.NodeConfig.StorageClass,
		Namespace: instance.ObjectMeta.Namespace,
	}
	if err := r.Client.Get(ctx, scNamespaceName, &sc); err != nil {
		if errors.IsNotFound(err) {
			addErrorCondition(err, instance, vmrayv1alpha1.NodeConfigInvalidStorageClass, vmrayv1alpha1.ResourceNotFoundReason)
		} else {
			r.Log.Error(err, "Failure when trying to fetch storage class.", "Namespace", scNamespaceName.Namespace, "Name", scNamespaceName.Name)
			return true, err
		}
		invalidState = true
	}

	// 3. Validate provided VM classes.
	vmclasses := []string{}
	for _, nt := range instance.Spec.NodeConfig.NodeTypes {
		vmclasses = append(vmclasses, nt.VMClass)
	}

	// remove duplicate elements from the vmclasses.
	vmclasses = slices.Compact(vmclasses)
	for i := range vmclasses {
		vmclass := vmopv1.VirtualMachineClass{}
		vmclassNamespaceName := types.NamespacedName{
			Name:      vmclasses[i],
			Namespace: instance.ObjectMeta.Namespace,
		}
		if err := r.Client.Get(ctx, vmclassNamespaceName, &vmclass); err != nil {
			if errors.IsNotFound(err) {
				addErrorCondition(err, instance, vmrayv1alpha1.NodeConfigInvalidVMClass, vmrayv1alpha1.ResourceNotFoundReason)
			} else {
				r.Log.Error(err, "Failure when trying to fetch VM class.", "Namespace", vmclassNamespaceName.Namespace, "Name", vmclasses[i])
				return true, err
			}
			invalidState = true
		}
	}
	return invalidState, nil
}

// createRandomNounce generates a random alpha-numeric string of given size.
func createRandomNounce(n int) string {
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		buf[i] = alphanumeric[rand.Intn(len(alphanumeric))]
	}
	return string(buf)
}
