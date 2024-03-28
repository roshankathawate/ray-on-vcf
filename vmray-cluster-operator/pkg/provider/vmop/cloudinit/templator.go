// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cloudinit

import (
	"bytes"
	"encoding/base64"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CloudConfigUserDataKey = "user-data"

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
  - su {{ (index .users 0).user }} -c 'sudo apt install python3-venv -y'
  - su {{ (index .users 0).user }} -c 'python3 -m venv /home/{{ (index .users 0).user }}/ray-env'
  - su {{ (index .users 0).user }} -c '/home/{{ (index .users 0).user }}/ray-env/bin/pip3 install pyvmomi'
  - su {{ (index .users 0).user }} -c '/home/{{ (index .users 0).user }}/ray-env/bin/pip3 install --upgrade git+https://github.com/vmware/vsphere-automation-sdk-python.git'
  - su {{ (index .users 0).user }} -c '/home/{{ (index .users 0).user }}/ray-env/bin/pip3 install -U "ray[data,train,tune,serve,default]"'
  - su {{ (index .users 0).user }} -c '/home/{{ (index .users 0).user }}/ray-env/bin/ray start --head --port=6379 --object-manager-port=8076 --autoscaling-config=~/ray_bootstrap_config.yaml --dashboard-host=0.0.0.0 > ~/ray_startup.log'
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
  - su {{ (index .users 0).user }} -c 'sudo apt install python3-venv -y'
  - su {{ (index .users 0).user }} -c 'python3 -m venv /home/{{ (index .users 0).user }}/ray-env'
  - su {{ (index .users 0).user }} -c '/home/{{ (index .users 0).user }}/ray-env/bin/pip3 install pyvmomi'
  - su {{ (index .users 0).user }} -c '/home/{{ (index .users 0).user }}/ray-env/bin/pip3 install --upgrade git+https://github.com/vmware/vsphere-automation-sdk-python.git'
  - su {{ (index .users 0).user }} -c '/home/{{ (index .users 0).user }}/ray-env/bin/pip3 install -U "ray[data,train,tune,serve,default]"'
  - su {{ (index .users 0).user }} -c '/home/{{ (index .users 0).user }}/ray-env/bin/ray start --address=$RAY_HEAD_IP:6379 --object-manager-port=8076 > ~/ray_worker_startup.log'
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

// produceCloudConfigYamlTemplate consumes user infos & files to mount to produce cloudinit configuration.
func produceCloudConfigYamlTemplate(users, files []map[string]string, headnode bool) ([]byte, error) {

	var templ *template.Template
	var err error

	if headnode {
		templ, err = template.New("cloud-config").Parse(cloudConfigHeadNodeTemplate)
	} else {
		templ, err = template.New("cloud-config").Parse(cloudConfigWorkerNodeTemplate)
	}

	if err != nil {
		return []byte{}, err
	}

	buf := bytes.NewBufferString("")
	templ.Execute(buf, map[string]interface{}{
		"users": users,
		"files": files,
	})
	return buf.Bytes(), nil
}

func CreateCloudConfigSecret(namespace, clusterName, secretName, svcAccToken, vmUser, vmPasswordHash string, headnode bool) (*corev1.Secret, error) {

	bootstrapYamlString, err := convertToYaml(getDefaultRayBootstrapConfig(clusterName), 5)
	if err != nil {
		return nil, err
	}

	supervisiorConfig, err := convertToYaml(supervisiorClusterConfig{
		ServiceAccountToken: svcAccToken,
	}, 5)
	if err != nil {
		return nil, err
	}

	users := []map[string]string{{
		"user":         vmUser,
		"passSaltHash": vmPasswordHash,
	}}

	var files []map[string]string
	if headnode {
		files = append(files,
			map[string]string{
				"path":    "/home/rayvm/ray_bootstrap_config.yaml",
				"content": bootstrapYamlString,
			},
			map[string]string{
				"path":    "/home/rayvm/supervisior-config.yaml",
				"content": supervisiorConfig,
			})
	}

	data, err := produceCloudConfigYamlTemplate(users, files, headnode)
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			CloudConfigUserDataKey: base64.StdEncoding.EncodeToString(data),
		},
	}, nil
}
