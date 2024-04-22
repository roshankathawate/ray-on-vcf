// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/internal/controller/lcm"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	DefaultRequeueDuration = time.Minute
	setupLog               = ctrl.Log.WithName("VMRayClusterReconciler")
	err                    error
)

const (
	finalizerName = "vmraycluster.vmray.broadcom.com"
	headsuffix    = "-head"
)

// VMRayClusterReconciler reconciles a VMRayCluster object
type VMRayClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	provider provider.VmProvider
	nlcm     *lcm.NodeLifecycleManager
}

func NewVMRayClusterReconciler(client client.Client, Scheme *runtime.Scheme, provider provider.VmProvider) *VMRayClusterReconciler {
	return &VMRayClusterReconciler{
		Client:   client,
		Scheme:   Scheme,
		provider: provider,
		nlcm:     lcm.NewNodeLifecycleManager(provider),
	}
}

//+kubebuilder:rbac:groups=vmray.broadcom.com,resources=vmrayclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vmray.broadcom.com,resources=vmrayclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vmray.broadcom.com,resources=vmrayclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VMRayCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *VMRayClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	setupLog.Info("Reconciling VMRayCluster", "cluster name", req.Name)

	// Fetch the RayCluster instance.
	instance := vmrayv1alpha1.VMRayCluster{}
	if err = r.Get(ctx, req.NamespacedName, &instance); err != nil {
		//Ignore not found errors
		if errors.IsNotFound(err) {
			setupLog.Info("Read request instance not found error!", "name", req.NamespacedName)
		} else {
			setupLog.Error(err, "Read request instance error!")
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If deletion timestamp is non-zero then execute delete reoncile loop.
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.VMRayClusterDelete(ctx, &instance); err != nil {
			return r.updateStatus(ctx, &instance)
		}
		return ctrl.Result{}, r.removeFinalizer(ctx, &instance)
	}

	return r.VMRayClusterReconcile(ctx, &instance)
}

// addFinalizer, this adds annotation during creation which will make
// k8s not delete the object until the set finalizer is removed.
// ref: https://sdk.operatorframework.io/docs/building-operators/golang/advanced-topics/#external-resources
func (r *VMRayClusterReconciler) addFinalizer(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) error {
	if instance.ObjectMeta.DeletionTimestamp.IsZero() &&
		!controllerutil.ContainsFinalizer(instance, finalizerName) {
		// add finalizer in case of create/update.
		_ = controllerutil.AddFinalizer(instance, finalizerName)
		setupLog.Info("VMRayCluster adding finalizer", "finalizer", finalizerName, "clustername", instance.Name)
		return r.Update(ctx, instance)
	}
	return nil
}

// removeFinalizer, this removes finalizer annotation so
// k8s can delete the mentioned CRD.
func (r *VMRayClusterReconciler) removeFinalizer(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) error {
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() &&
		controllerutil.ContainsFinalizer(instance, finalizerName) {
		_ = controllerutil.RemoveFinalizer(instance, finalizerName)
		setupLog.Info("VMRayCluster removing finalizer", "finalizer", finalizerName, "clustername", instance.Name)
		return r.Update(ctx, instance)
	}
	return nil
}

func (r *VMRayClusterReconciler) VMRayClusterReconcile(
	ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) (ctrl.Result, error) {

	// Step 0: [TODO] figure out standard way to tracks observed conditions.
	instance.Status.Conditions = []metav1.Condition{}

	// Step 1: Check if it's create request, if so add finalizer.
	if err := r.addFinalizer(ctx, instance); err != nil {
		return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, err
	}

	// Step 2: Reconcile head node.
	if err := r.reconcileHeadNode(ctx, instance); err != nil {
		instance.Status.ClusterState = vmrayv1alpha1.UNHEALTHY
		setupLog.Error(err, "VMRayCluster reconcile head failed", "cluster name", instance.Name)

		// Update status to show head node failure as observed condition.
		addErrorCondition(err, instance, vmrayv1alpha1.VMRayClusterConditionHeadNodeReady, vmrayv1alpha1.FailureToDeployNodeReason)

		// TODO: Set error message as to why cluster state is unhealthy.
		return r.updateStatus(ctx, instance)
	}

	// Make sure ray process in head node is running
	// successfully, before reconciling worker nodes.
	if instance.Status.HeadNodeStatus.RayStatus != vmrayv1alpha1.RAY_RUNNING {
		return r.updateStatus(ctx, instance)
	}

	// Step 3: Reconcile worker nodes.
	if err := r.reconcileWorkerNodes(ctx, instance); err != nil {
		instance.Status.ClusterState = vmrayv1alpha1.UNHEALTHY
		setupLog.Error(err, "VMRayCluster reconcile worker node failed", "cluster name", instance.Name)

		// Update status to show worker node failure as observed condition.
		addErrorCondition(err, instance, vmrayv1alpha1.VMRayClusterConditionWorkerNodeReady, vmrayv1alpha1.FailureToDeployNodeReason)
	} else {
		instance.Status.ClusterState = vmrayv1alpha1.HEALTHY
	}

	// Step 4: Update the Ray cluster instance status.
	return r.updateStatus(ctx, instance)
}

func (r *VMRayClusterReconciler) VMRayClusterDelete(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) error {
	setupLog.Info("Entering reconcile vmraycluster delete", "clustername", instance.Name)

	// Step 1: Delete service account, role, role bindings & secret.
	err = r.provider.DeleteAuxiliaryResources(ctx, instance.Namespace, instance.Name)
	if err != nil {
		setupLog.Error(err, "Failure when trying to delete auxiliary resources.", "cluster name", instance.Name)
		addErrorCondition(err, instance, vmrayv1alpha1.VMRayClusterConditionClusterDelete, vmrayv1alpha1.FailureToDeleteAuxiliaryResourcesReason)
		return err
	}

	// Step 2: Delete all worker nodes.
	err = r.deleteWorkerNodes(ctx, instance, true)
	if err != nil {
		setupLog.Error(err, "Failure when trying to delete worker nodes.", "cluster name", instance.Name)
		addErrorCondition(err, instance, vmrayv1alpha1.VMRayClusterConditionClusterDelete, vmrayv1alpha1.FailureToDeleteHeadNodeReason)
		return err
	}

	// Step 3: Delete head node.
	headNodeName := instance.Name + headsuffix
	err = r.provider.Delete(ctx, instance.Namespace, headNodeName)
	if err != nil {
		setupLog.Error(err, "Failure when trying to delete head node.", "cluster name", instance.Name)
		addErrorCondition(err, instance, vmrayv1alpha1.VMRayClusterConditionClusterDelete, vmrayv1alpha1.FailureToDeleteWorkerNodeReason)
		return err
	}

	setupLog.Info("Successfully deleted vmraycluster instance.", "clustername", instance.Name)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VMRayClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vmrayv1alpha1.VMRayCluster{}).
		Complete(r)
}
