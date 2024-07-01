// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/internal/controller/lcm"
	vmprovider "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	finalizerName              = "vmraycluster.vmray.broadcom.com"
	HeadNodeNounceLabel        = "vmray.kubernetes.io/head-nounce"
	nouceLength            int = 5
	DefaultRequeueDuration     = 60 * time.Second
)

// VMRayClusterReconciler reconciles a VMRayCluster object
type VMRayClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	provider vmprovider.VmProvider
	nlcm     *lcm.NodeLifecycleManager
}

func NewVMRayClusterReconciler(client client.Client, Scheme *runtime.Scheme, provider vmprovider.VmProvider) *VMRayClusterReconciler {
	return &VMRayClusterReconciler{
		Client:   client,
		Scheme:   Scheme,
		provider: provider,
		nlcm:     lcm.NewNodeLifecycleManager(provider),
		Log:      ctrl.Log.WithName("VMRayClusterReconciler"),
	}
}

type reconcileEnvelope struct {
	OriginalClusterState *vmrayv1alpha1.VMRayCluster
	CurrentClusterState  *vmrayv1alpha1.VMRayCluster
}

//+kubebuilder:rbac:groups=vmray.broadcom.com,resources=vmrayclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vmray.broadcom.com,resources=vmrayclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vmray.broadcom.com,resources=vmrayclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// The Reconcile function compares the state specified by
// the VMRayCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *VMRayClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Reconciling VMRayCluster", "cluster name", req.NamespacedName)

	// Fetch the latest VMRayCluster instance.
	instance := vmrayv1alpha1.VMRayCluster{}
	if err := r.fetchVMRayCluster(ctx, req.NamespacedName, &instance); err != nil {
		// Error reading the object - requeue the request.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	re := reconcileEnvelope{
		CurrentClusterState:  &instance,
		OriginalClusterState: instance.DeepCopy(),
	}

	// If deletion timestamp is non-zero then execute delete reoncile loop.
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.VMRayClusterDelete(ctx, re.CurrentClusterState); err != nil {
			return r.updateStatus(ctx, re)
		}
		return ctrl.Result{}, r.removeFinalizer(ctx, re.CurrentClusterState)
	}

	return r.VMRayClusterReconcile(ctx, re)
}

// addFinalizerAndNounce, this adds Finalizer annotation during creation which
// will make sure k8s doesn't delete the object until the set finalizer is removed.
// ref: https://sdk.operatorframework.io/docs/building-operators/golang/advanced-topics/#external-resources
//
// This function also adds unique nounce which will be head node's identifier
// to overcome bug [PROT-625], where on re-application if head VM from previous
// request still exists it won't face the already exists error.
func (r *VMRayClusterReconciler) addFinalizerAndNounce(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) error {
	if instance.ObjectMeta.DeletionTimestamp.IsZero() &&
		!controllerutil.ContainsFinalizer(instance, finalizerName) {
		// Add finalizer in case of create/update.
		_ = controllerutil.AddFinalizer(instance, finalizerName)
		r.Log.Info("VMRayCluster adding finalizer", "finalizer", finalizerName, "clustername", instance.ObjectMeta.Name)

		// Add nouce label to the cluster.
		if instance.ObjectMeta.Labels == nil {
			instance.ObjectMeta.Labels = make(map[string]string)
		}
		instance.ObjectMeta.Labels[HeadNodeNounceLabel] = createRandomNounce(nouceLength)

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
		r.Log.Info("VMRayCluster removing finalizer", "finalizer", finalizerName, "clustername", instance.Name)
		return r.Update(ctx, instance)
	}
	return nil
}

func (r *VMRayClusterReconciler) VMRayClusterReconcile(
	ctx context.Context, re reconcileEnvelope) (ctrl.Result, error) {

	// Step 1: Check if it's create request, if so add finalizer.
	instance := re.CurrentClusterState
	if err := r.addFinalizerAndNounce(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Step 2: Reconcile head node.
	instance.Status.Conditions = []metav1.Condition{}
	instance.Status.ClusterState = vmrayv1alpha1.HEALTHY

	if err := r.reconcileHeadNode(ctx, instance); err != nil {
		instance.Status.ClusterState = vmrayv1alpha1.UNHEALTHY
		r.Log.Error(err, "VMRayCluster reconcile head failed", "cluster name", instance.Name)

		// Update status to show head node failure as observed condition.
		addErrorCondition(err, instance, vmrayv1alpha1.VMRayClusterConditionHeadNodeReady, vmrayv1alpha1.FailureToDeployNodeReason)
	}

	// Make sure ray process in head node is running
	// successfully, before reconciling worker nodes.
	if instance.Status.HeadNodeStatus.RayStatus == vmrayv1alpha1.RAY_RUNNING {
		// Step 3: Reconcile worker nodes.
		if err := r.reconcileWorkerNodes(ctx, instance); err != nil {
			r.Log.Error(err, "VMRayCluster reconcile worker node failed", "cluster name", instance.Name)
			// Mark cluster state as unhealthy if we fail to create atleast minimum workers
			if uint((len(instance.Status.CurrentWorkers))) <= instance.Spec.WorkerNode.MinWorkers {
				instance.Status.ClusterState = vmrayv1alpha1.UNHEALTHY
				// Update status to show worker node failure as observed condition.
				addErrorCondition(err, instance, vmrayv1alpha1.VMRayClusterConditionWorkerNodeReady, vmrayv1alpha1.FailureToDeployNodeReason)
			}
		}
	}

	// Step 4: Update the Ray cluster instance status.
	return r.updateStatus(ctx, re)
}

func (r *VMRayClusterReconciler) VMRayClusterDelete(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster) error {
	r.Log.Info("Entering reconcile vmraycluster delete", "clustername", instance.Name)

	// Step 1: Delete service account, role, role bindings & secret.
	err := r.provider.DeleteAuxiliaryResources(ctx, instance.Namespace, instance.Name)
	if err != nil {
		r.Log.Error(err, "Failure when trying to delete auxiliary resources.", "cluster name", instance.Name)
		addErrorCondition(err, instance, vmrayv1alpha1.VMRayClusterConditionClusterDelete, vmrayv1alpha1.FailureToDeleteAuxiliaryResourcesReason)
		return err
	}

	// Step 2: Delete all worker nodes.
	err = r.deleteWorkerNodes(ctx, instance, true)
	if err != nil {
		r.Log.Error(err, "Failure when trying to delete worker nodes.", "cluster name", instance.ObjectMeta.Name)
		addErrorCondition(err, instance, vmrayv1alpha1.VMRayClusterConditionClusterDelete, vmrayv1alpha1.FailureToDeleteWorkerNodeReason)
		return err
	}

	// Step 3: Delete head node.
	nounce := instance.ObjectMeta.Labels[HeadNodeNounceLabel]
	err = r.provider.Delete(ctx, instance.ObjectMeta.Namespace, vmprovider.GetHeadNodeName(instance.ObjectMeta.Name, nounce))
	if err != nil {
		r.Log.Error(err, "Failure when trying to delete head node.", "cluster name", instance.ObjectMeta.Name)
		addErrorCondition(err, instance, vmrayv1alpha1.VMRayClusterConditionClusterDelete, vmrayv1alpha1.FailureToDeleteHeadNodeReason)
		return err
	}

	r.Log.Info("Successfully deleted vmraycluster instance.", "clustername", instance.ObjectMeta.Name)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VMRayClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vmrayv1alpha1.VMRayCluster{}).
		Complete(r)
}
