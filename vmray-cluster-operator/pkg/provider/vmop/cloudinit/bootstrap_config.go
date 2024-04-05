// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	"time"
)

type RayBootstrapConfig struct {
	ClusterName                string            `yaml:"cluster_name"`
	MaxWorkers                 int               `yaml:"max_workers"`
	UpscalingSpeed             int               `yaml:"upscaling_speed"`
	Docker                     docker            `yaml:"docker"`
	IdleTimeoutMinutes         time.Duration     `yaml:"idle_timeout_minutes"`
	Provider                   provider          `yaml:"provider"`
	Auth                       auth              `yaml:"auth"`
	AvailableNodeTypes         map[string]node   `yaml:"available_node_types"`
	HeadNodeType               string            `yaml:"head_node_type"`
	FileMounts                 map[string]string `yaml:"file_mounts"`
	ClusterSyncedFiles         []string          `yaml:"cluster_synced_files"`
	FileMountsSyncContinuously bool              `yaml:"file_mounts_sync_continuously"`
	RsyncExclude               []string          `yaml:"rsync_exclude"`
	RsyncFilter                []string          `yaml:"rsync_filter"`
	InitializationCommands     []string          `yaml:"initialization_commands"`
	SetupCommands              []string          `yaml:"setup_commands"`
	HeadSetupCommands          []string          `yaml:"head_setup_commands"`
	WorkerSetupCommands        []string          `yaml:"worker_setup_commands"`
	HeadStartRayCommands       []string          `yaml:"head_start_ray_commands"`
	WorkerStartRayCommands     []string          `yaml:"worker_start_ray_commands"`
	NoRestart                  bool              `yaml:"no_restart"`
}

type docker struct {
	Image         string   `yaml:"image"`
	ContainerName string   `yaml:"container_name"`
	PullBeforeRun bool     `yaml:"pull_before_run"`
	RunOptions    []string `yaml:"run_options"`
}
type credentials struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Server   string `yaml:"server"`
}
type frozenVM struct {
	Name string `yaml:"name"`
}
type vsphereConfig struct {
	Credentials credentials `yaml:"credentials"`
	FrozenVM    frozenVM    `yaml:"frozen_vm"`
}
type provider struct {
	Type              string        `yaml:"type"`
	CacheStoppedNodes bool          `yaml:"cache_stopped_nodes"`
	VsphereConfig     vsphereConfig `yaml:"vsphere_config"`
}
type auth struct {
	SSHUser       string `yaml:"ssh_user"`
	SSHPrivateKey string `yaml:"ssh_private_key"`
}

type resources struct {
}

type nodeConfig struct {
	Resources    resources `yaml:"resources"`
	ResourcePool string    `yaml:"string"`
	FrozenVM     frozenVM  `yaml:"frozen_vm"`
}

type node struct {
	NodeConfig nodeConfig `yaml:"node_config"`
	MinWorkers int        `yaml:"min_workers"`
	MaxWorkers int        `yaml:"max_workers"`
	Resources  resources  `yaml:"resources"`
}

func getDefaultRayBootstrapConfig(cloudConfig CloudConfig) RayBootstrapConfig {
	return RayBootstrapConfig{
		ClusterName:    cloudConfig.VmDeploymentRequest.ClusterName,
		MaxWorkers:     2,
		UpscalingSpeed: 1,
		Docker: docker{
			Image:         cloudConfig.VmDeploymentRequest.DockerImage,
			ContainerName: "ray_container",
			PullBeforeRun: false,
			RunOptions:    []string{"--ulimit nofile=65536:65536"},
		},
		IdleTimeoutMinutes: 5 * time.Minute,
		Provider: provider{
			Type:              "vsphere",
			CacheStoppedNodes: true,
			VsphereConfig: vsphereConfig{
				Credentials: credentials{
					User:     "<vsphere-username>",
					Password: "<vsphere-password>",
					Server:   "<x.x.x.x>",
				},
				FrozenVM: frozenVM{
					Name: "frozen",
				},
			},
		},
		Auth: auth{
			SSHUser:       "ray",
			SSHPrivateKey: "~/ray_bootstrap.key.pem",
		},
		AvailableNodeTypes: map[string]node{
			"ray.head.default": {
				Resources: resources{},
				NodeConfig: nodeConfig{
					Resources: resources{},
					FrozenVM: frozenVM{
						Name: "frozen",
					},
				},
				MinWorkers: 0,
				MaxWorkers: 0,
			},
			"worker": {
				Resources: resources{},
				NodeConfig: nodeConfig{
					Resources: resources{},
					FrozenVM: frozenVM{
						Name: "frozen",
					},
				},
				MinWorkers: 0,
				MaxWorkers: 3,
			},
		},
		HeadNodeType:               "ray.head.default",
		FileMounts:                 map[string]string{},
		ClusterSyncedFiles:         []string{},
		FileMountsSyncContinuously: false,
		RsyncExclude:               []string{"**/.git", "**/.git/**"},
		RsyncFilter:                []string{".gitignore"},
		InitializationCommands:     []string{},
		SetupCommands:              []string{},
		HeadSetupCommands:          []string{},
		WorkerSetupCommands:        []string{},
		HeadStartRayCommands: []string{
			"ulimit -n 65536; ray start --head --port=6379 --object-manager-port=8076" +
				" --autoscaling-config=~/ray_bootstrap_config.yaml --dashboard-host=0.0.0.0",
		},
		WorkerStartRayCommands: []string{
			"ulimit -n 65536; ray start --address=$RAY_HEAD_IP:6379 --object-manager-port=8076",
		},
		NoRestart: false,
	}
}
