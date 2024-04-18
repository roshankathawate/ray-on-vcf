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
	Image string `json:"image,omitempty"`
	// If the worker node stays idle for this time then bring it down.
	IdleTimeoutMinutes uint `json:"idle_timeout_minutes,omitempty"`

	// setup_commands sections provides the commands to be executed in the Ray's docker container prior to starting the Ray processes
	SetupCommands []string `json:"setup_commands,omitempty"`
	// Configuration for bringing up a jupyterhub environment
	JupyterHub *JupyterHubConfig `json:"jupyterhub,omitempty"`
	// Configuration for bringing up a Prometheus/Grafana environment.
	Monitoring *MonitoringConfig `json:"monitoring,omitempty"`
	// Configuration for the head node.
	HeadNode HeadNodeConfig `json:"head_node"`
	// Configuration for each of the worker nodes.
	WorkerNode WorkerNodeConfig `json:"worker_node"`
	// The desired names of workers. This field is only updated by the autoscaler.
	DesiredWorkers []string `json:"desired_workers,omitempty"` //This field will only be updated by the autoscaler so we can omit if it's not specified by the user.
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
	INITIALIZED VMNodeStatus = "initialized"
	RUNNING     VMNodeStatus = "running"
	FAIL        VMNodeStatus = "failure"

	RAY_INITIALIZED RayProcessStatus = "initialized"
	RAY_RUNNING     RayProcessStatus = "running"
	RAY_FAIL        RayProcessStatus = "failure"
)

type VMRayNodeStatus struct {
	// VirtualMachine name is format of : clustername-[worker/head]-[uuid]
	Name string `json:"name,omitempty"`
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

type VMRayClusterState string

const (
	HEALTHY   VMRayClusterState = "healty"
	UNHEALTHY VMRayClusterState = "unhealthy"
)

// VMRayClusterStatus defines the observed state of VMRayCluster
type VMRayClusterStatus struct {
	// Status of ray head node.
	HeadNodeStatus VMRayNodeStatus `json:"head_node_status,omitempty"`
	// Statuses of each of the current workers
	CurrentWorkers []VMRayNodeStatus `json:"current_workers,omitempty"`
	// Overall state of the Ray cluster
	ClusterState VMRayClusterState `json:"cluster_state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VMRayCluster is the Schema for the vmrayclusters API
type VMRayCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The configuration of the Ray cluster
	Spec VMRayClusterSpec `json:"spec,omitempty"`
	// The Ray cluster status
	Status VMRayClusterStatus `json:"status,omitempty"`
}

type JupyterHubConfig struct {
	// The docker image for jupyterhub
	Image string `json:"image,omitempty"`
	// The user can provide a premium docker account credentials to avoid rate limiting.
	DockerCredsSecret string `json:"docker_creds_secret,omitempty"`
}

type MonitoringConfig struct {
	// The docker image for prometheus
	PrometheusImage string `json:"prometheus_image,omitempty"`
	// The docker image for grafana
	GrafanaImage string `json:"grafana_image,omitempty"`
	// The user can provide a premium docker account credentials to avoid rate limiting.
	DockerCredsSecret string `json:"docker_creds_secret,omitempty"`
}

type HeadNodeConfig struct {
	// The VMRayNodeConfig CR contains the configuration of the VM.
	NodeConfigName string `json:"node_config_name"`
	// The setup commands are executed in Ray container before starting ray processes.
	HeadSetupCommands []string `json:"head_setup_commands,omitempty"`
}

type WorkerNodeConfig struct {
	// The VMRayNodeConfig CR contains the configuration of the VM.
	NodeConfigName string `json:"node_config_name"`
	// The setup commands are executed in Ray container before starting ray processes.
	WorkerSetupCommands []string `json:"worker_setup_commands,omitempty"`
	// The minimum number of workers
	MinWorkers uint `json:"min_workers"`
	// The maximum number of workers
	MaxWorkers uint `json:"max_workers"`
}

//+kubebuilder:object:root=true

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
