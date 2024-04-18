// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"
	"regexp"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var (
	nameRegex, _         = regexp.Compile("^[a-z]([-a-z0-9]*[a-z0-9])?$")
	dnsComplaintRegex, _ = regexp.Compile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
)

// log is for logging in this package.
var vmraynodeconfiglog = logf.Log.WithName("vmraynodeconfig-resource")

func (r *VMRayNodeConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-vmray-broadcom-com-v1alpha1-vmraynodeconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=vmray.broadcom.com,resources=vmraynodeconfigs,verbs=create;update,versions=v1alpha1,name=mvmraynodeconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &VMRayNodeConfig{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *VMRayNodeConfig) Default() {
	vmraynodeconfiglog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

//+kubebuilder:webhook:path=/validate-vmray-broadcom-com-v1alpha1-vmraynodeconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=vmray.broadcom.com,resources=vmraynodeconfigs,verbs=create;update,versions=v1alpha1,name=vvmraynodeconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &VMRayNodeConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *VMRayNodeConfig) ValidateCreate() (admission.Warnings, error) {
	vmraynodeconfiglog.Info("validate create VMRayNodeConfig new", "name", r.Name)

	return nil, r.validateVMRayNodeConfig()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *VMRayNodeConfig) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	vmraynodeconfiglog.Info("validate update VMRayNodeConfig", "name", r.Name)

	return nil, r.validateVMRayNodeConfig()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *VMRayNodeConfig) ValidateDelete() (admission.Warnings, error) {
	// TODO: Implement deletion validation if required.

	return nil, nil
}

func (r *VMRayNodeConfig) validateVMRayNodeConfig() error {
	var allErrs field.ErrorList

	if err := r.validateName(); err != nil {
		allErrs = append(allErrs, err)
	}

	// 1. Maximum 63 chars allowed
	// 2. Must be DNS complaint.
	if err := r.validateVirtualMachineClassName(); err != nil {
		allErrs = append(allErrs, err)
	}

	// Up to 80 characters may be used for content library name
	if err := r.validateCharLimit(field.NewPath("spec").Child("content_library"), r.Spec.ContentLibrary); err != nil {
		allErrs = append(allErrs, err)
	}

	// Up to 80 characters may be used for OVF name
	if err := r.validateCharLimit(field.NewPath("spec").Child("ovf"), r.Spec.Ovf); err != nil {
		allErrs = append(allErrs, err)
	}

	// Up to 80 characters may be used for Storage policy name
	if err := r.validateCharLimit(field.NewPath("spec").Child("storage_policy"), r.Spec.StoragePolicy); err != nil {
		allErrs = append(allErrs, err)
	}

	// Up to 80 characters may be used for Network policy name
	if err := r.validateCharLimit(field.NewPath("spec").Child("network_policy"), r.Spec.NetworkPolicy); err != nil {
		allErrs = append(allErrs, err)
	}

	// 1. Maximum 253 characters can be specified for cloud init config.
	// 2. A lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character
	if err := r.validateCloudInitConfig(); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(schema.GroupKind{Group: "vmray", Kind: "VMRayNodeConfig"}, r.Name, allErrs)
}

func (r *VMRayNodeConfig) validateName() *field.Error {
	if !nameRegex.MatchString(r.Name) {
		return field.Invalid(field.NewPath("metadata").Child("name"), r.Name, fmt.Sprintf("Must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character. Regex used for validation is '%s'", nameRegex))
	}
	return nil
}

func (r *VMRayNodeConfig) validateVirtualMachineClassName() *field.Error {
	var dnsNameLen = len(r.Spec.VMClass)

	if dnsNameLen > 63 {
		return field.Invalid(field.NewPath("spec").Child("vm_class"), r.Spec.VMClass, "Maximum 63 characters are allowed")
	}

	// Must be DNS complaint.
	if !dnsComplaintRegex.MatchString(r.Spec.VMClass) {
		return field.Invalid(field.NewPath("spec").Child("vm_class"), r.Spec.VMClass, fmt.Sprintf("Must be DNS complaint. The regex used is '%s'", dnsComplaintRegex))
	}

	return nil
}

func (r *VMRayNodeConfig) validateCharLimit(fieldPath *field.Path, fieldName string) *field.Error {
	// Up to 80 characters may be used
	if len(fieldName) > 80 {
		return field.Invalid(fieldPath, fieldName, "Maximum 80 characters are allowed.")
	}

	return nil
}

func (r *VMRayNodeConfig) validateCloudInitConfig() *field.Error {
	vmraynodeconfiglog.Info("validate cloud init config", "name", r.Name)
	var cloudInitConfigLen = len(r.Spec.CloudInitConfig)
	var fieldPath = field.NewPath("spec").Child("cloud_init_config")

	if cloudInitConfigLen > 253 {
		return field.Invalid(fieldPath, r.Spec.CloudInitConfig, "Maximum of 253 characters are allowed.")
	}

	if !dnsComplaintRegex.MatchString(r.Spec.CloudInitConfig) {
		return field.Invalid(fieldPath, r.Spec.CloudInitConfig, fmt.Sprintf("Must be DNS complaint. The regex used is '%s'", dnsComplaintRegex))
	}

	return nil
}
