// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (

	// Conditions which could be observed by VMRayNodeConfig reconciler.
	VMRayNodeConfigInvalidVMI          = "InvalidVirtualMachineImage"
	VMRayNodeConfigInvalidStorageClass = "InvalidStorageClass"
	VMRayNodeConfigInvalidVMClass      = "InvalidVirtualMachineClass"

	// List of reasons for the observed conditions.
	VMRayNodeConfigResourceNotFoundReason = "ResourceNotFound"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VMRayNodeConfigSpec defines the desired state of VMRayNodeConfig
type VMRayNodeConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The VM class for Ray nodes
	VMClass string `json:"vm_class"`
	// Name of VirtualMachineImage of type ovf used to create ray nodes i.e. mapped against content library item.
	VMImage string `json:"vm_image"`
	// The NFS location that should be mounted to the Ray nodes.
	Nfs string `json:"nfs,omitempty"`
	// Storage class associated with for a specific namespace in supervisor cluster.
	StorageClass string `json:"storage_class"`
	// Name of user space that we should create to run Ray Process in VM.
	VMUser string `json:"vm_user"`
	// Value of password's SHA-512 salt hash to be set for provided user name in ray VM.
	VMPasswordSaltHash string `json:"vm_password_salt_hash"`
	// Network describes the desired network configuration for the VM.
	// +optional
	Network *vmopv1.VirtualMachineNetworkSpec `json:"network,omitempty"`
}

// VMRayNodeConfigStatus defines the observed state of VMRayNodeConfig
type VMRayNodeConfigStatus struct {
	// Valid represents if nodeconfig CR is of valid state.
	Valid *bool `json:"Valid,omitempty"`

	// Conditions describes the observed conditions of the VMRayNodeConfig.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VMRayNodeConfig is the Schema for the vmraynodeconfigs API
type VMRayNodeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// The configuration of the VMRayVirtualMachine
	Spec VMRayNodeConfigSpec `json:"spec,omitempty"`
	// The current status of the VMRayVirtualMachine
	Status VMRayNodeConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VMRayNodeConfigList contains a list of VMRayNodeConfig
type VMRayNodeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VMRayNodeConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VMRayNodeConfig{}, &VMRayNodeConfigList{})
}
