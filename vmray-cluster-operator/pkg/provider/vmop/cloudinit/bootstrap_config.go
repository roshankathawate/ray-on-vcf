// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit

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
	CaCert    string `yaml:"ca_cert"`
	APIServer string `yaml:"api_server"`
	Namespace string `yaml:"namespace"`
}
type Provider struct {
	Type          string        `yaml:"type"`
	VsphereConfig VsphereConfig `yaml:"vsphere_config"`
}
type Docker struct {
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
}
type NodeConfig struct {
}
type Node struct {
	NodeConfig NodeConfig `yaml:"node_config"`
	MinWorkers uint       `yaml:"min_workers"`
	MaxWorkers uint       `yaml:"max_workers"`
	Resources  Resources  `yaml:"resources"`
}

const (
	ProviderType   = "vsphere"
	UpscalingSpeed = 1
)

func getRayBootstrapConfig(cloudConfig CloudConfig) RayBootstrapConfig {
	return RayBootstrapConfig{
		ClusterName:        cloudConfig.VmDeploymentRequest.ClusterName,
		MaxWorkers:         cloudConfig.VmDeploymentRequest.WorkerNodeConfig.MaxWorkers,
		MinWorkers:         cloudConfig.VmDeploymentRequest.WorkerNodeConfig.MinWorkers,
		UpscalingSpeed:     UpscalingSpeed,
		Docker:             Docker{},
		IdleTimeoutMinutes: cloudConfig.VmDeploymentRequest.WorkerNodeConfig.IdleTimeoutMinutes,
		Provider: Provider{
			Type: ProviderType,
			VsphereConfig: VsphereConfig{
				CaCert:    cloudConfig.VmDeploymentRequest.ApiServer.CaCert,
				APIServer: cloudConfig.VmDeploymentRequest.ApiServer.Location,
				Namespace: cloudConfig.VmDeploymentRequest.Namespace,
			},
		},
		Auth: Auth{},
		AvailableNodeTypes: map[string]Node{
			"ray.head.default": {
				NodeConfig: NodeConfig{},
				Resources:  Resources{},
			},
			"worker": {
				NodeConfig: NodeConfig{},
				MinWorkers: cloudConfig.VmDeploymentRequest.WorkerNodeConfig.MinWorkers,
				MaxWorkers: cloudConfig.VmDeploymentRequest.WorkerNodeConfig.MaxWorkers,
				Resources:  Resources{},
			},
		},
		HeadNodeType:               "ray.head.default",
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
