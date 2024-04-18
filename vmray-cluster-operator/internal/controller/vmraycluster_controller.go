// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/internal/controller/lcm"
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
)

var (
	DefaultRequeueDuration = time.Minute
	setupLog               = ctrl.Log.WithName("VMRayClusterReconciler")
	err                    error
)

const (
	headsuffix = "-head"
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

	instance := &vmrayv1alpha1.VMRayCluster{}

	// Fetch the RayCluster instance
	if err = r.Get(ctx, req.NamespacedName, instance); err != nil {
		//Ignore not found errors
		if errors.IsNotFound(err) {
			setupLog.Info("Read request instance not found error!", "name", req.NamespacedName)
		} else {
			setupLog.Error(err, "Read request instance error!")
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if instance.DeletionTimestamp != nil && !instance.DeletionTimestamp.IsZero() {
		//TODO: logic to delete Ray cluster to be filled here
		setupLog.Info("VMRayCluster is being deleted, just ignore", "cluster name", req.Name)
		return ctrl.Result{}, nil
	}
	return r.VMRayClusterReconcile(ctx, req, instance)
}

func (r *VMRayClusterReconciler) VMRayClusterReconcile(
	ctx context.Context, request ctrl.Request, instance *vmrayv1alpha1.VMRayCluster) (ctrl.Result, error) {

	// Step 1: Reconcile head node.
	if err := r.reconcileHeadNode(ctx, instance, request); err != nil {
		instance.Status.ClusterState = vmrayv1alpha1.UNHEALTHY
		setupLog.Error(err, "VMRayCluster reconcile head failed", "cluster name", request.Name)

		// TODO: Set error message as to why cluster state is unhealthy.

		return r.updateStatus(ctx, instance, request.Name)
	}

	// Step 2: Reconcile worker nodes.

	// Step 2.1: Figure our workers which need to be deleted, submit request to remove them.

	// Step 2.2: Figure out list of new set of workers that need to be added.
	// Leverage node lifecycle manager to process the VM state.

	// Step 3: Update the Ray cluster instance status.
	return r.updateStatus(ctx, instance, request.Name)
}

func (r *VMRayClusterReconciler) reconcileHeadNode(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster, req ctrl.Request) error {
	setupLog.Info("Reconciling head node.")

	// Step 1: Get Node config required by head node.
	nodeConfig, err := r.getNodeConfig(ctx, req.Namespace, instance.Spec.HeadNode.NodeConfigName)
	if err != nil {
		setupLog.Error(err, "Failure to get head node config", "name", instance.Spec.HeadNode.NodeConfigName)
		return err
	}

	headNodeName := instance.Name + headsuffix
	// Step 2: leverage node lifecycle manager to process headnode state.
	err = r.nlcm.ProcessNodeVmState(ctx, req.Namespace, instance.Name, headNodeName, nodeConfig, &instance.Status.HeadNodeStatus)
	if err != nil {
		return err
	}

	// [TODO] Step 3: process Ray status in VM.
	return nil
}

func (r *VMRayClusterReconciler) getNodeConfig(ctx context.Context, namespace, nodeConfigName string) (*vmrayv1alpha1.VMRayNodeConfig, error) {
	nodeConfig := &vmrayv1alpha1.VMRayNodeConfig{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      nodeConfigName,
	}
	//get the nodeconfig object
	if err = r.Get(ctx, key, nodeConfig); err != nil {
		return nil, err
	}
	return nodeConfig, nil
}

func (r *VMRayClusterReconciler) updateStatus(ctx context.Context, instance *vmrayv1alpha1.VMRayCluster, reqName string) (ctrl.Result, error) {
	setupLog.Info("rayClusterReconcile", "Update Ray cluster CR status", reqName, "status", instance.Status)
	err := r.Status().Update(ctx, instance)
	if err != nil {
		setupLog.Info("Got error when updating status", "cluster name", reqName, "error", err, "RayCluster", instance)
	}
	return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *VMRayClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vmrayv1alpha1.VMRayCluster{}).
		Complete(r)
}
