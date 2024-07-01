// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmrayv1alpha1 "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/api/v1alpha1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VMRayNodeConfigReconciler reconciles a VMRayNodeConfig object
type VMRayNodeConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func NewVMRayNodeConfigReconciler(client client.Client, Scheme *runtime.Scheme) *VMRayNodeConfigReconciler {
	return &VMRayNodeConfigReconciler{
		Client: client,
		Scheme: Scheme,
		Log:    ctrl.Log.WithName("VMRayNodeConfigReconciler"),
	}
}

//+kubebuilder:rbac:groups=vmray.broadcom.com,resources=vmraynodeconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vmray.broadcom.com,resources=vmraynodeconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vmray.broadcom.com,resources=vmraynodeconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VMRayNodeConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *VMRayNodeConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	r.Log.Info("Reconciling VMRayNodeConfig", "Node config", req.NamespacedName)

	// Fetch the latest VMRayNodeConfig instance.
	instance := vmrayv1alpha1.VMRayNodeConfig{}
	if err := r.fetchVMRayNodeConfig(ctx, req.NamespacedName, &instance); err != nil {
		// Error reading the object - requeue the request.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Reset observed conditions for new reconcle loop.
	noErrorDetected := true
	originalVMRayNodeConfig := instance.DeepCopy()
	instance.Status.Conditions = []metav1.Condition{}

	// Validate existence of VMI, storage class and
	// vm classs associated with-in namespace.
	// 1. Validate VM image
	vmi := vmopv1.VirtualMachineImage{}
	vmiNamespaceName := types.NamespacedName{
		Name:      instance.Spec.VMImage,
		Namespace: instance.ObjectMeta.Namespace,
	}
	if err := r.Get(ctx, vmiNamespaceName, &vmi); err != nil {
		noErrorDetected = false
		//Ignore not found errors
		if errors.IsNotFound(err) {
			addNodeConfigErrorCondition(err, &instance, vmrayv1alpha1.VMRayNodeConfigInvalidVMI, vmrayv1alpha1.VMRayNodeConfigResourceNotFoundReason)
		} else {
			r.Log.Error(err, "Failure when trying to fetch VM image.", "Namespace", vmiNamespaceName.Namespace, "Name", vmiNamespaceName.Name)
		}
	}

	// 2. Validate Storage class.
	sc := storagev1.StorageClass{}
	scNamespaceName := types.NamespacedName{
		Name:      instance.Spec.StorageClass,
		Namespace: instance.ObjectMeta.Namespace,
	}
	if err := r.Get(ctx, scNamespaceName, &sc); err != nil {
		noErrorDetected = false
		//Ignore not found errors
		if errors.IsNotFound(err) {
			addNodeConfigErrorCondition(err, &instance, vmrayv1alpha1.VMRayNodeConfigInvalidStorageClass, vmrayv1alpha1.VMRayNodeConfigResourceNotFoundReason)
		} else {
			r.Log.Error(err, "Failure when trying to fetch storage class.", "Namespace", scNamespaceName.Namespace, "Name", scNamespaceName.Name)
		}
	}

	// 3. Validate VM class.
	vmclass := vmopv1.VirtualMachineClass{}
	vmclassNamespaceName := types.NamespacedName{
		Name:      instance.Spec.VMClass,
		Namespace: instance.ObjectMeta.Namespace,
	}
	if err := r.Get(ctx, vmclassNamespaceName, &vmclass); err != nil {
		noErrorDetected = false
		//Ignore not found errors
		if errors.IsNotFound(err) {
			addNodeConfigErrorCondition(err, &instance, vmrayv1alpha1.VMRayNodeConfigInvalidVMClass, vmrayv1alpha1.VMRayNodeConfigResourceNotFoundReason)
		} else {
			r.Log.Error(err, "Failure when trying to fetch VM class.", "Namespace", vmclassNamespaceName.Namespace, "Name", vmclassNamespaceName.Name)
		}
	}

	if !noErrorDetected {
		r.Log.Info("Detected errors for node config inputs", "Node config", req.NamespacedName, "Error conditions", instance.Status.Conditions)
	}
	// Set validity of nodeconfig resource.
	instance.Status.Valid = &noErrorDetected

	return r.updateStatus(ctx, &instance, originalVMRayNodeConfig)
}

// SetupWithManager sets up the controller with the Manager.
func (r *VMRayNodeConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vmrayv1alpha1.VMRayNodeConfig{}).
		Complete(r)
}

func (r *VMRayNodeConfigReconciler) fetchVMRayNodeConfig(ctx context.Context, namespacedName types.NamespacedName, instance *vmrayv1alpha1.VMRayNodeConfig) error {
	var err error
	if err = r.Get(ctx, namespacedName, instance); err != nil {
		// Ignore not found errors.
		if errors.IsNotFound(err) {
			r.Log.Error(err, "Read request instance not found", "name", namespacedName)
		} else {
			r.Log.Error(err, "Read request instance error")
		}
	}
	return err
}

func (r *VMRayNodeConfigReconciler) updateStatus(ctx context.Context, current, original *vmrayv1alpha1.VMRayNodeConfig) (ctrl.Result, error) {
	name := current.ObjectMeta.Name
	status := current.Status

	if reflect.DeepEqual(current.Status.Conditions, original.Status.Conditions) {
		r.Log.Info("No change to node config status", "name", name, "status", status)
		return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, nil
	}
	r.Log.Info("Update Ray node config CR status", "name", name, "current", status, "original", original.Status)

	patch := client.MergeFrom(original)
	err := r.Client.Status().Patch(ctx, current, patch)
	if err != nil {
		r.Log.Error(err, "Error when updating status", "NodeConfig name", name, "current", current)
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, nil
}

func addNodeConfigErrorCondition(err error, instance *vmrayv1alpha1.VMRayNodeConfig, Type, Reason string) {
	instance.Status.Conditions = append(instance.Status.Conditions, metav1.Condition{
		Type:               Type,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Reason:             Reason,
		Message:            err.Error(),
	})
}
