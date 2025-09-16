// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (

	// Conditions which could be observed by our operators.
	VMRayClusterConditionHeadNodeReady   = "HeadNodeReady"
	VMRayClusterConditionWorkerNodeReady = "WorkerNodeReady"
	VMRayClusterConditionClusterDelete   = "DeleteCluster"

	// Conditions which could be observed by  reconciler.
	NodeConfigInvalidVMI          = "InvalidVirtualMachineImage"
	NodeConfigInvalidStorageClass = "InvalidStorageClass"
	NodeConfigInvalidVMClass      = "InvalidVirtualMachineClass"

	// List of reasons for the observed conditions.
	FailureToDeployNodeReason               = "FailureToDeployNode"
	FailureToDeleteAuxiliaryResourcesReason = "FailureToDeleteAuxiliaryResources"
	FailureToDeleteHeadNodeReason           = "FailureToDeleteHeadNode"
	FailureToDeleteWorkerNodeReason         = "FailureToDeleteWorkerNode"
	ResourceNotFoundReason                  = "ResourceNotFound"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VMRayClusterSpec defines the desired state of VMRayCluster
type VMRayClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// image holds name of ray's image needed during cluster deployment.
	Image string `json:"ray_docker_image"`
	// api_server holds information needed on API server.
	ApiServer ApiServerInfo `json:"api_server"`
	// Configuration for the head node.
	HeadNode HeadNodeConfig `json:"head_node"`
	// Configuration for the worker node.
	WorkerNode WorkerNodeConfig `json:"worker_node"`
	// This defines the common configuration of each VM i.e. ray head or worker node.
	NodeConfig CommonNodeConfig `json:"common_node_config"`
	// The desired names & config of workers. This field is only updated by the autoscaler.
	AutoscalerDesiredWorkers map[string]string `json:"autoscaler_desired_workers,omitempty"` // This field will only be updated by the autoscaler so we can omit if it's not specified by the user.
	// Enable/Disable TLS on Ray gRPC channels
	// +kubebuilder:default=true
	// +optional
	EnableTLS bool `json:"enable_tls"`
	// This defines node's docker's configuration, such as authentication details with registry.
	DockerConfig DockerRegistryConfig `json:"docker_config,omitempty"`
}

type VMNodeStatus string
type RayProcessStatus string

/*
Process flow:
 1. If VM status string is empty, that means user just created & submitted the CRD.
    we perfom deploy activity and set both VM & ray process status to `initialized`.
 2. In next reconcile cycle, when request comes in with status set to `initialized`
 	 we check VM CRD for IP assignment. Based on it's availability, we set VM status
	 to either `running` state or leave it in `initialized` state. Ray process is
	 validated in similar manner independent of VM status.
 3. If previous state was running, and VM IP doesnt exists or is not reachable or
 	 if ray status is unhealthy. Then we set the status to failure accordingly.
*/

const (
	EMPTY       VMNodeStatus = ""
	INITIALIZED VMNodeStatus = "initialized"
	RUNNING     VMNodeStatus = "running"
	FAIL        VMNodeStatus = "failure"

	RAY_INITIALIZED RayProcessStatus = "initialized"
	RAY_RUNNING     RayProcessStatus = "running"
	RAY_FAIL        RayProcessStatus = "failure"
)

type VMRayNodeStatus struct {
	// Observed primary IP of VirtualMachine.
	Ip string `json:"ip,omitempty"`
	// Conditions describes the observed conditions of the VirtualMachine.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// This will define & track VM status.
	VmStatus VMNodeStatus `json:"vm_status,omitempty"`
	// This will define & track ray process status.
	RayStatus RayProcessStatus `json:"ray_status,omitempty"`
}

type VMServiceStatus struct {
	// IP captures first ingress IP of vm service
	// associated with head VirtualMachine.
	Ip string `json:"ip,omitempty"`
}

type VMRayClusterState string

const (
	HEALTHY   VMRayClusterState = "healthy"
	UNHEALTHY VMRayClusterState = "unhealthy"
)

// VMRayClusterStatus defines the observed state of VMRayCluster
type VMRayClusterStatus struct {
	// Status of ray head node.
	HeadNodeStatus VMRayNodeStatus `json:"head_node_status,omitempty"`
	// Statuses of each of the current workers
	CurrentWorkers map[string]VMRayNodeStatus `json:"current_workers,omitempty"`
	// Overall state of the Ray cluster
	ClusterState VMRayClusterState `json:"cluster_state,omitempty"`
	// Conditions describes the observed conditions of the VMRayCluster.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Status of VM service associated with head VirtualMachine.
	VMServiceStatus VMServiceStatus `json:"vm_service_status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VMRayCluster is the Schema for the vmrayclusters API
type VMRayCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The configuration of the Ray cluster
	Spec VMRayClusterSpec `json:"spec,omitempty"`
	// The Ray cluster status
	Status VMRayClusterStatus `json:"status,omitempty"`
}

type ApiServerInfo struct {
	// ca_cert holds base64 value of CA cert of API server.
	CaCert string `json:"ca_cert,omitempty"`
	// location holds IP or domain name of supervisor cluster's master node.
	Location string `json:"location"`
}

type HeadNodeConfig struct {
	// These setup commands are executed in head node's Ray container before starting ray process.
	SetupCommands []string `json:"setup_commands,omitempty"`
	// The Port specifies port of the head ray process running in VM.
	// +optional
	Port *uint `json:"port"`
	// NodeType represents key for one of the node types in available_node_types.
	// This node type will be used to launch the head node.
	NodeType string `json:"node_type"`
}

type WorkerNodeConfig struct {
	// These setup commands are executed in worker node's Ray container before starting ray process.
	SetupCommands []string `json:"setup_commands,omitempty"`
}

type DockerRegistryConfig struct {
	// Used to pass name of secret containing information regarding registry credentials.
	AuthSecretName string `json:"auth_secret_name"`
}

type CommonNodeConfig struct {
	// Name of VirtualMachineImage of type ovf used to create ray nodes i.e. mapped against content library item.
	VMImage string `json:"vm_image"`
	// Storage class associated with for a specific namespace in supervisor cluster.
	StorageClass string `json:"storage_class"`
	// Name of user space that we should create to run Ray Process in VM.
	VMUser string `json:"vm_user"`
	// Value of password's SHA-512 salt hash to be set for provided user name in ray VM.
	VMPasswordSaltHash string `json:"vm_password_salt_hash"`
	// Network describes the desired network configuration for the VM.
	// +optional
	Network *vmopv1.VirtualMachineNetworkSpec `json:"network,omitempty"`
	// Node types describe type of ray node configuration that can be deployed.
	NodeTypes map[string]NodeType `json:"available_node_types"`
	// The maximum number of workers
	MaxWorkers uint `json:"max_workers"`
	// If the worker node stays idle for this time then bring it down.
	IdleTimeoutMinutes uint `json:"idle_timeout_minutes,omitempty"`
	// These are common setup commands executed in Ray container before starting ray process in both head & worker nodes.
	SetupCommands []string `json:"setup_commands,omitempty"`
	// These commands will run outside the container in ray's VM node before docker container starts.
	InitializationCommands []string `json:"initialization_commands,omitempty"`
}

type NodeType struct {
	// The VM class for Ray nodes
	VMClass string `json:"vm_class"`
	// The minimum number of workers
	MinWorkers uint `json:"min_workers"`
	// The maximum number of workers
	MaxWorkers uint `json:"max_workers"`
	// Resource limit to be set to be leveraged by ray process towards workload
	Resources NodeResource `json:"resources,omitempty"`
}

type NodeResource struct {
	// CPU limit to be used by the node.
	CPU uint8 `json:"cpu,omitempty"`

	// Memory limit to be used by the node.
	Memory uint `json:"memory,omitempty"`

	// GPU limit to be used by the node.
	GPU uint8 `json:"gpu,omitempty"`
}

// +kubebuilder:object:root=true

// VMRayClusterList contains a list of VMRayCluster
type VMRayClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// The list of ray clusters
	Items []VMRayCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VMRayCluster{}, &VMRayClusterList{})
}
