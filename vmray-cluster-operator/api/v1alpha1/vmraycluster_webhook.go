// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"
	"regexp"

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
	nameRegex, _         = regexp.Compile("^[a-z]([-a-z0-9]*[a-z0-9])?$")
	dnsComplaintRegex, _ = regexp.Compile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	vmrayclusterlog      = logf.Log.WithName("vmraycluster-resource")
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
	vmrayclusterlog.Info("default", "name", r.ObjectMeta.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-vmray-broadcom-com-v1alpha1-vmraycluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=vmray.broadcom.com,resources=vmrayclusters,verbs=create;update,versions=v1alpha1,name=vvmraycluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &VMRayCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *VMRayCluster) ValidateCreate() (admission.Warnings, error) {
	vmrayclusterlog.Info("validate create", "name", r.ObjectMeta.Name)

	return nil, r.validateVMRayCluster()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *VMRayCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	vmrayclusterlog.Info("validate update", "name", r.ObjectMeta.Name)

	return nil, r.validateVMRayCluster()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *VMRayCluster) ValidateDelete() (admission.Warnings, error) {
	vmrayclusterlog.Info("validate delete", "name", r.ObjectMeta.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

func (r *VMRayCluster) validateVMRayCluster() error {
	var allErrs field.ErrorList

	if err := r.validateName(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateMinMax(); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateMinNodeTypes(); err != nil {
		allErrs = append(allErrs, err)
	}

	// Validate Ray docker image
	if err := r.validateDockerImage(field.NewPath("spec").Child("image"), r.Spec.Image); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := r.validateDesiredWorkers(field.NewPath("spec").Child("autoscaler_desired_workers")); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "ray.io", Kind: "RayCluster"},
		r.ObjectMeta.Name, allErrs)
}

func (r *VMRayCluster) validateName() *field.Error {
	if !nameRegex.MatchString(r.ObjectMeta.Name) {
		return field.Invalid(field.NewPath("metadata").Child("name"),
			r.ObjectMeta.Name, "name must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')")
	}
	return nil
}

func (r *VMRayCluster) validateMinMax() *field.Error {
	// Validate it for cluster level.
	if r.Spec.NodeConfig.MinWorkers > r.Spec.NodeConfig.MaxWorkers {
		return field.Invalid(field.NewPath("spec").Child("common_node_config"), "min_workers/max_workers",
			fmt.Sprintf("Min workers cannot be more than Max workers, min_workers: %d, max_workers: %d",
				r.Spec.NodeConfig.MinWorkers, r.Spec.NodeConfig.MaxWorkers))
	}

	// Validate for each node type.
	total_min_worker := uint(0)
	for name, nt := range r.Spec.NodeConfig.NodeTypes {
		if nt.MinWorkers > nt.MaxWorkers {
			return field.Invalid(field.NewPath("spec").Child("common_node_config").Child("node_types").Child(name), "min_workers/max_workers",
				fmt.Sprintf("Min workers cannot be more than Max workers, min_workers: %d, max_workers: %d",
					nt.MinWorkers, nt.MaxWorkers))
		}

		if nt.MaxWorkers > r.Spec.NodeConfig.MaxWorkers {
			return field.Invalid(field.NewPath("spec").Child("common_node_config").Child("node_types").Child(name), "max_workers",
				fmt.Sprintf("Available node type max worker count shoudn't be more than cluster max_worker count, node_type: %s, spec.common_node_config.max_workers: %d",
					name, r.Spec.NodeConfig.MaxWorkers))

		}
		total_min_worker = total_min_worker + nt.MinWorkers
	}
	if total_min_worker > r.Spec.NodeConfig.MinWorkers {
		return field.Invalid(field.NewPath("spec").Child("common_node_config"), "min_workers",
			fmt.Sprintf("Expected spec.common_node_config.min_workers is: %d, but total desired min worker for all available node type is: %d",
				r.Spec.NodeConfig.MinWorkers, total_min_worker))
	}
	return nil
}

func (r *VMRayCluster) validateMinNodeTypes() *field.Error {
	if len(r.Spec.NodeConfig.NodeTypes) > 0 {
		return nil
	}
	return field.Invalid(field.NewPath("spec").Child("common_node_config"), "node_types", "Should atleast have one or more node config")
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
	for k := range r.Spec.AutoscalerDesiredWorkers {
		if !dnsComplaintRegex.MatchString(k) {
			return field.Invalid(fieldPath, "name",
				fmt.Sprintf("Must be DNS compliant name %s", k))
		}
	}
	return nil
}
