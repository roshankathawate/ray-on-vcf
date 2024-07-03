// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"

	"github.com/distribution/reference"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	vmrayclusterlog = logf.Log.WithName("vmraycluster-resource")
)

func (r *VMRayCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-vmray-broadcom-com-v1alpha1-vmraycluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=vmray.broadcom.com,resources=vmrayclusters,verbs=create;update,versions=v1alpha1,name=mvmraycluster.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &VMRayCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *VMRayCluster) Default() {
	vmrayclusterlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-vmray-broadcom-com-v1alpha1-vmraycluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=vmray.broadcom.com,resources=vmrayclusters,verbs=create;update,versions=v1alpha1,name=vvmraycluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &VMRayCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *VMRayCluster) ValidateCreate() (admission.Warnings, error) {
	vmrayclusterlog.Info("validate create", "name", r.Name)

	return nil, r.validateVMRayCluster()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *VMRayCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	vmrayclusterlog.Info("validate update", "name", r.Name)

	return nil, r.validateVMRayCluster()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *VMRayCluster) ValidateDelete() (admission.Warnings, error) {
	vmrayclusterlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

func (r *VMRayCluster) validateVMRayCluster() error {
	var allErrs field.ErrorList

	if err := r.validateName(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateHeadNodeConfig(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateWorkerNodeConfig(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateMinMaxWorkers(); err != nil {
		allErrs = append(allErrs, err)
	}
	// Validate Ray docker image
	if err := r.validateDockerImage(field.NewPath("spec").Child("image"), r.Spec.Image); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateDesiredWorkers(field.NewPath("spec").Child("DesiredWorkers")); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "ray.io", Kind: "RayCluster"},
		r.Name, allErrs)
}

func (r *VMRayCluster) validateName() *field.Error {
	if !nameRegex.MatchString(r.Name) {
		return field.Invalid(field.NewPath("metadata").Child("name"),
			r.Name, "name must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')")
	}
	return nil
}

func (r *VMRayCluster) validateHeadNodeConfig() *field.Error {
	if r.Spec.HeadNode.NodeConfigName == "" {
		return field.Invalid(field.NewPath("spec").Child("HeadNode").Child("NodeConfig"), "NodeConfig",
			"NodeConfig is a required field in HeadNodeConfig")
	}
	return nil
}

func (r *VMRayCluster) validateWorkerNodeConfig() *field.Error {
	if r.Spec.WorkerNode.NodeConfigName == "" {
		return field.Invalid(field.NewPath("spec").Child("WorkerNode").Child("NodeConfig"), "NodeConfig",
			"NodeConfig is a required field in WorkerNodeConfig")
	}
	return nil
}

func (r *VMRayCluster) validateMinMaxWorkers() *field.Error {
	vmrayclusterlog.Info("validate min_max_worker", "min_worker", r.Spec.WorkerNode.MinWorkers)
	vmrayclusterlog.Info("validate min_max_worker", "max_worker", r.Spec.WorkerNode.MaxWorkers)
	if r.Spec.WorkerNode.MinWorkers > r.Spec.WorkerNode.MaxWorkers {
		vmrayclusterlog.Info("validate min_max_worker", "name", r.Name)
		return field.Invalid(field.NewPath("spec").Child("worker_node"), "min_workers/max_workers",
			fmt.Sprintf("Min workers cannot be more than Max workers, min_workers: %d, max_workers: %d",
				r.Spec.WorkerNode.MinWorkers, r.Spec.WorkerNode.MaxWorkers))
	}
	return nil
}

func (r *VMRayCluster) validateDockerImage(fieldPath *field.Path, dockerImageName string) *field.Error {
	vmrayclusterlog.Info("dockerImageName", "name", dockerImageName)
	if !reference.ReferenceRegexp.MatchString(dockerImageName) {
		vmrayclusterlog.Info("Regex passed", "image", dockerImageName)
		return field.Invalid(fieldPath, dockerImageName,
			fmt.Sprintf("Docker image is invalid. The regex used is '%s'", reference.ReferenceRegexp))
	}
	return nil
}

func (r *VMRayCluster) validateDesiredWorkers(fieldPath *field.Path) *field.Error {
	for i := 0; i < len(r.Spec.DesiredWorkers); i++ {
		if !dnsComplaintRegex.MatchString(r.Spec.DesiredWorkers[i]) {
			return field.Invalid(fieldPath, "name",
				fmt.Sprintf("Must be DNS compliant name %s", r.Spec.DesiredWorkers[i]))
		}
	}
	return nil
}
