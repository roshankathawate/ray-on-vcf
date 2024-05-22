// Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	vmprovider "gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CloudConfig struct {
	VmDeploymentRequest    vmprovider.VmDeploymentRequest
	SvcAccToken            string
	SecretName             string
	HeadVmServiceIngressIp string
}

const (
	CloudInitConfigUserDataKey = "user-data"
	RayHeadDefaultPort         = int32(6379)
	RayHeadStartCmd            = "ray start --head --port=%d --block --autoscaling-config=/home/ray/ray_bootstrap_config.yaml --dashboard-host=0.0.0.0"

	// Templates to generate cloud config for Ray head & worker nodes.
	cloudConfigHeadNodeTemplate = `#cloud-config
ssh_pwauth: true
users:
{{- range .users }}
  - name: {{ .user }}
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: false
    passwd: "{{ .passSaltHash }}"
    shell: /bin/bash
{{- end }}
write_files:
{{- range .files }}
  - content: |
{{ .content }}
    path: {{ .path }}
{{- end }}
runcmd:
  - chown -R {{ (index .users 0).user }}:{{ (index .users 0).user }} /home/{{ (index .users 0).user }}
  - usermod -aG docker {{ (index .users 0).user }}
  - su {{ (index .users 0).user }} -c 'apt-get update && apt-get install -y docker'
  - su {{ (index .users 0).user }} -c 'docker pull {{ .docker_image }}'
  - su {{ (index .users 0).user }} -c 'docker run --rm --name ray_container -d -v {{ (index .files 0).path }}:/home/ray/ray_bootstrap_config.yaml -p {{ .ray_head_port }}:{{ .ray_head_port }} --env "SVC_ACCOUNT_TOKEN={{ .svc_account_token }}" {{ .docker_image }}  /bin/bash -c "{{ .docker_cmd }}"'
`

	cloudConfigWorkerNodeTemplate = `#cloud-config
ssh_pwauth: true
users:
{{- range .users }}
  - name: {{ .user }}
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: false
    passwd: "{{ .passSaltHash }}"
    shell: /bin/bash
{{- end }}
runcmd:
  - chown -R {{ (index .users 0).user }}:{{ (index .users 0).user }} /home/{{ (index .users 0).user }}
  - usermod -aG docker {{ (index .users 0).user }}
  - su {{ (index .users 0).user }} -c 'apt-get update && apt-get install -y docker'
  - su {{ (index .users 0).user }} -c 'docker pull {{ .docker_image }}'
  - su {{ (index .users 0).user }} -c 'docker run --rm --name ray_container -d --network host {{ .docker_image }} /bin/bash -c "ray start --block --address={{ .head_vmservice_ingress_ip }}:{{ .ray_head_port }}"'
`
)

// addIndentation is used to add required padding to file contents injected in cloud config.
func addIndentation(yaml string, indentation int) string {
	if indentation <= 0 {
		return yaml
	}

	indentationPrefix := strings.Repeat(" ", indentation)

	out := []string{}
	split := strings.SplitAfter(yaml, "\n")
	for i := 0; i < len(split); i++ {
		out = append(out, indentationPrefix+split[i])
	}
	return strings.Join(out, "")
}

// convertToYaml converts provided interface to its corresponding yaml reflection.
func convertToYaml(config interface{}, indentation int) (string, error) {
	b, err := yaml.Marshal(&config)
	if err != nil {
		return "", err
	}
	return addIndentation(string(b), indentation), nil
}

// produceCloudInitConfigYamlTemplate consumes user infos & files to mount to produce cloudinit configuration.
func produceCloudInitConfigYamlTemplate(cloudConfig CloudConfig) ([]byte, error) {

	var templ *template.Template
	var err error

	bootstrapYamlString, err := convertToYaml(getRayBootstrapConfig(cloudConfig), 5)
	if err != nil {
		return nil, err
	}

	vmuser := cloudConfig.VmDeploymentRequest.NodeConfigSpec.VMUser
	users := []map[string]string{{
		"user":         vmuser,
		"passSaltHash": cloudConfig.VmDeploymentRequest.NodeConfigSpec.VMPasswordSaltHash,
	}}

	if cloudConfig.VmDeploymentRequest.HeadNodeStatus == nil {
		templ, err = template.New("cloud-config").Parse(cloudConfigHeadNodeTemplate)
	} else {
		templ, err = template.New("cloud-config").Parse(cloudConfigWorkerNodeTemplate)
	}

	if err != nil {
		return []byte{}, err
	}

	var files []map[string]string

	// HeadNodeStatus will be nil if it is the head node.
	var port = RayHeadDefaultPort
	if cloudConfig.VmDeploymentRequest.HeadNodeConfig.Port != nil {
		port = int32(*cloudConfig.VmDeploymentRequest.HeadNodeConfig.Port)
	}

	var docker_cmd = ""
	if cloudConfig.VmDeploymentRequest.HeadNodeStatus == nil {
		files = append(files,
			map[string]string{
				"path":    fmt.Sprintf("/home/%s/ray_bootstrap_config.yaml", vmuser),
				"content": bootstrapYamlString,
			},
		)
		setup_cmds := append(cloudConfig.VmDeploymentRequest.HeadNodeConfig.SetupCommands, fmt.Sprintf(RayHeadStartCmd, port))
		docker_cmd = strings.Join(setup_cmds, ";")
	}

	buf := bytes.NewBufferString("")
	if err = templ.Execute(buf, map[string]interface{}{
		"users":                     users,
		"files":                     files,
		"docker_image":              cloudConfig.VmDeploymentRequest.DockerImage,
		"head_vmservice_ingress_ip": cloudConfig.HeadVmServiceIngressIp,
		"svc_account_token":         cloudConfig.SvcAccToken,
		"docker_cmd":                docker_cmd,
		"ray_head_port":             port,
	}); err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func CreateCloudInitConfigSecret(cloudConfig CloudConfig) (*corev1.Secret, error) {

	data, err := produceCloudInitConfigYamlTemplate(cloudConfig)
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      cloudConfig.SecretName,
			Namespace: cloudConfig.VmDeploymentRequest.Namespace,
		},
		StringData: map[string]string{
			CloudInitConfigUserDataKey: base64.StdEncoding.EncodeToString(data),
		},
	}, nil
}
