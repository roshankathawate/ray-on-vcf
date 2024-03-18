// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VMRayNodeConfigSpec defines the desired state of VMRayNodeConfig
type VMRayNodeConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The VM class for head node
	VmClass string `json:"vm_class"`
	// Name of the Content Library where the OVF resides.
	ContentLibrary string `json:"content_library"`
	// The OVF file from which the Ray cluster's nodes will be created.
	Ovf string `json:"ovf"`
	// The NFS location that should be mounted to the Ray nodes.
	Nfs string `json:"nfs,omitempty"`
	// Storage policy
	StoragePolicy string `json:"storage_policy"`
	// Network policy
	NetworkPolicy string `json:"network_policy,omitempty"`
	// Config map name that stores the cloud init config
	CloudInitConfig string `json:"cloud_init_config,omitempty"`
}

// VMRayNodeConfigStatus defines the observed state of VMRayNodeConfig
type VMRayNodeConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
