// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VMRayClusterSpec defines the desired state of VMRayCluster
type VMRayClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Image holds name of ray's image needed during cluster deployment.
	Image string `json:"image"`
}

// VMRayClusterStatus defines the observed state of VMRayCluster
type VMRayClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VMRayCluster is the Schema for the vmrayclusters API
type VMRayCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VMRayClusterSpec   `json:"spec,omitempty"`
	Status VMRayClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VMRayClusterList contains a list of VMRayCluster
type VMRayClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VMRayCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VMRayCluster{}, &VMRayClusterList{})
}
