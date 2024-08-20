// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/constants"
)

type RayBootstrapConfig struct {
	ClusterName                string          `yaml:"cluster_name"`
	MaxWorkers                 uint            `yaml:"max_workers"`
	MinWorkers                 uint            `yaml:"min_workers"`
	UpscalingSpeed             int             `yaml:"upscaling_speed"`
	Docker                     Docker          `yaml:"docker"`
	IdleTimeoutMinutes         uint            `yaml:"idle_timeout_minutes"`
	Provider                   Provider        `yaml:"provider"`
	Auth                       Auth            `yaml:"auth"`
	AvailableNodeTypes         map[string]Node `yaml:"available_node_types"`
	HeadNodeType               string          `yaml:"head_node_type"`
	FileMounts                 FileMounts      `yaml:"file_mounts"`
	ClusterSyncedFiles         []string        `yaml:"cluster_synced_files"`
	FileMountsSyncContinuously bool            `yaml:"file_mounts_sync_continuously"`
	RsyncExclude               []string        `yaml:"rsync_exclude"`
	RsyncFilter                []string        `yaml:"rsync_filter"`
	InitializationCommands     []string        `yaml:"initialization_commands"`
	SetupCommands              []string        `yaml:"setup_commands"`
	HeadSetupCommands          []string        `yaml:"head_setup_commands"`
	WorkerSetupCommands        []string        `yaml:"worker_setup_commands"`
	HeadStartRayCommands       []string        `yaml:"head_start_ray_commands"`
	WorkerStartRayCommands     []string        `yaml:"worker_start_ray_commands"`
	NoRestart                  bool            `yaml:"no_restart"`
}
type VsphereConfig struct {
	CaCert       string `yaml:"ca_cert"`
	APIServer    string `yaml:"api_server"`
	Namespace    string `yaml:"namespace"`
	VmImage      string `yaml:"vm_image"`
	StorageClass string `yaml:"storage_policy"`
}
type Provider struct {
	Type          string        `yaml:"type"`
	VsphereConfig VsphereConfig `yaml:"vsphere_config"`
}
type Docker struct {
	Image            string   `yaml:"image"`
	ContainerName    string   `yaml:"container_name"`
	pullBeforeRun    bool     `yaml:"pull_before_run"`
	RunOptions       []string `yaml:"run_options"`
	WorkerRunOptions []string `yaml:"worker_run_options"`
}
type Auth struct {
	SSHUser   string `yaml:"ssh_user"`
	SSHPvtKey string `yaml:"ssh_private_key"`
}
type RayHeadDefault struct {
}
type Worker struct {
}
type AvailableNodeTypes struct {
}
type FileMounts struct {
}
type Resources struct {
	CPU    uint8 `json:"cpu"`
	Memory uint  `json:"memory"`
	GPU    uint8 `json:"gpu"`
}
type NodeConfig struct {
	VMclass string `yaml:"vmclass"`
}
type Node struct {
	NodeConfig NodeConfig `yaml:"node_config"`
	MinWorkers uint       `yaml:"min_workers"`
	MaxWorkers uint       `yaml:"max_workers"`
	Resources  Resources  `yaml:"resources"`
}

func getRayBootstrapConfig(cloudConfig CloudConfig) RayBootstrapConfig {
	return RayBootstrapConfig{
		ClusterName:    cloudConfig.VmDeploymentRequest.ClusterName,
		MaxWorkers:     cloudConfig.VmDeploymentRequest.NodeConfig.MaxWorkers,
		MinWorkers:     cloudConfig.VmDeploymentRequest.NodeConfig.MinWorkers,
		UpscalingSpeed: constants.UpscalingSpeed,
		Docker: Docker{
			Image:            cloudConfig.VmDeploymentRequest.DockerImage,
			ContainerName:    ray_container_name,
			pullBeforeRun:    true,
			RunOptions:       []string{},
			WorkerRunOptions: []string{},
		},
		IdleTimeoutMinutes: cloudConfig.VmDeploymentRequest.NodeConfig.IdleTimeoutMinutes,
		Provider: Provider{
			Type: constants.ProviderType,
			VsphereConfig: VsphereConfig{
				CaCert:       cloudConfig.VmDeploymentRequest.ApiServer.CaCert,
				APIServer:    cloudConfig.VmDeploymentRequest.ApiServer.Location,
				Namespace:    cloudConfig.VmDeploymentRequest.Namespace,
				VmImage:      cloudConfig.VmDeploymentRequest.NodeConfig.VMImage,
				StorageClass: cloudConfig.VmDeploymentRequest.NodeConfig.StorageClass,
			},
		},
		Auth: Auth{
			SSHUser:   cloudConfig.VmDeploymentRequest.NodeConfig.VMUser,
			SSHPvtKey: constants.SSHPvtKeyPath,
		},
		AvailableNodeTypes:         getAvailableNodeTypes(cloudConfig),
		HeadNodeType:               constants.DefaultHeadNodeType,
		FileMounts:                 FileMounts{},
		ClusterSyncedFiles:         []string{},
		FileMountsSyncContinuously: false,
		RsyncExclude:               []string{"**/.git", "**/.git/**"},
		RsyncFilter:                []string{".gitignore"},
		InitializationCommands:     []string{},
		SetupCommands:              []string{},
		HeadSetupCommands:          []string{},
		WorkerSetupCommands:        []string{},
		HeadStartRayCommands:       []string{},
		WorkerStartRayCommands:     []string{},
		NoRestart:                  false,
	}
}

func getAvailableNodeTypes(cloudConfig CloudConfig) map[string]Node {
	availabletypes := map[string]Node{}

	for key, nt := range cloudConfig.VmDeploymentRequest.NodeConfig.NodeTypes {
		availabletypes[key] = Node{
			MinWorkers: nt.MinWorkers,
			MaxWorkers: nt.MaxWorkers,
			Resources: Resources{
				CPU:    nt.Resources.CPU,
				Memory: nt.Resources.Memory,
				GPU:    nt.Resources.GPU,
			},
			NodeConfig: NodeConfig{
				VMclass: nt.VMClass,
			},
		}
	}
	return availabletypes
}
