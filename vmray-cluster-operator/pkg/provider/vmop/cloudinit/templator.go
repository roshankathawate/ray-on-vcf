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
	"gitlab.eng.vmware.com/xlabs/x77-taiga/vmray/vmray-cluster-operator/pkg/provider/vmop/tls"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CloudConfig struct {
	VmDeploymentRequest vmprovider.VmDeploymentRequest
	SvcAccToken         string
	SecretName          string
	SshPvtKey           string
	CaCrt               string
	CaKey               string
	HeadNodeIp          string
	EnableTLS           int
}

const (
	CloudInitConfigUserDataKey = "user-data"
	SshPrivateKey              = "ssh-pvt-key"
	ssh_rsa_key_file           = "id_rsa_ray"
	Ca_cert_file               = "ca.crt"
	Ca_key_file                = "ca.key"
	ssh_rsa_key_path_in_docker = "/home/ray/.ssh/id_rsa_ray"
	RayHeadDefaultPort         = int32(6379)
	RayHeadStartCmd            = "ray start --head --port=%d --block --autoscaling-config=/home/ray/ray_bootstrap_config.yaml --dashboard-host=0.0.0.0"
	ray_bootstrap_config_file  = "ray_bootstrap_config.yaml"
	RunScriptToGenCerts        = "sh /home/ray/gencert.sh"

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
{{- if .permissions }}
    permissions: '{{ .permissions }}'
{{- end }}
{{- end }}
runcmd:
  - chown -R {{ (index .users 0).user }}:{{ (index .users 0).user }} /home/{{ (index .users 0).user }}
  - usermod -aG docker {{ (index .users 0).user }}
  - su {{ (index .users 0).user }} -c 'ssh-keygen -f {{ .ssh_rsa_key_path }} -t RSA -y > {{ .ssh_rsa_key_path }}.pub'
  - su {{ (index .users 0).user }} -c 'cat {{ .ssh_rsa_key_path }}.pub >> ~/.ssh/authorized_keys'
  - su {{ (index .users 0).user }} -c 'apt-get update && apt-get install -y docker'
  - su {{ (index .users 0).user }} -c 'docker pull {{ .docker_image }}'
  - su {{ (index .users 0).user }} -c 'docker run --rm --name ray_container -d --network host -v {{ .ray_bootstrap_config_file_path }}:/home/ray/ray_bootstrap_config.yaml -v {{ .tls_ca_cert_path }}:/home/ray/ca.crt -v {{ .tls_ca_key_path }}:/home/ray/ca.key -v {{ .gen_cert_script_path }}:/home/ray/gencert.sh -v {{ .ssh_rsa_key_path }}:/home/ray/.ssh/id_rsa_ray --env "SVC_ACCOUNT_TOKEN={{ .svc_account_token }}" --env "RAY_USE_TLS={{ .enable_tls }}" --env "RAY_TLS_CA_CERT=/home/ray/ca.crt" --env "RAY_TLS_SERVER_KEY=/home/ray/tls.key" --env "RAY_TLS_SERVER_CERT=/home/ray/tls.crt" {{ .docker_image }}  /bin/bash -c "sudo -i -u root chmod 0777 /home/ray/.ssh/id_rsa_ray; {{ .setup_cmd }}; {{ .docker_cmd }}"'
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
write_files:
{{- range .files }}
  - content: |
{{ .content }}
    path: {{ .path }}
{{- if .permissions }}
    permissions: '{{ .permissions }}'
{{- end }}
{{- end }}
runcmd:
  - chown -R {{ (index .users 0).user }}:{{ (index .users 0).user }} /home/{{ (index .users 0).user }}
  - usermod -aG docker {{ (index .users 0).user }}
  - su {{ (index .users 0).user }} -c 'apt-get update && apt-get install -y docker'
  - su {{ (index .users 0).user }} -c 'ssh-keygen -f {{ .ssh_rsa_key_path }} -t RSA -y > {{ .ssh_rsa_key_path }}.pub'
  - su {{ (index .users 0).user }} -c 'cat {{ .ssh_rsa_key_path }}.pub >> ~/.ssh/authorized_keys'
  - su {{ (index .users 0).user }} -c 'docker pull {{ .docker_image }}'
  - su {{ (index .users 0).user }} -c 'docker run --rm --name ray_container -d --network host -v {{ .tls_ca_cert_path }}:/home/ray/ca.crt -v {{ .tls_ca_key_path }}:/home/ray/ca.key -v {{ .gen_cert_script_path }}:/home/ray/gencert.sh --env "RAY_USE_TLS={{ .enable_tls }}" --env "RAY_TLS_CA_CERT=/home/ray/ca.crt" --env "RAY_TLS_SERVER_KEY=/home/ray/tls.key" --env "RAY_TLS_SERVER_CERT=/home/ray/tls.crt" {{ .docker_image }} /bin/bash -c "{{ .setup_cmd }}; ray start --block --address={{ .head_node_ip }}:{{ .ray_head_port }}"'
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

	vmuser := cloudConfig.VmDeploymentRequest.NodeConfig.VMUser
	users := []map[string]string{{
		"user":         vmuser,
		"passSaltHash": cloudConfig.VmDeploymentRequest.NodeConfig.VMPasswordSaltHash,
	}}
	ssh_rsa_key_path := fmt.Sprintf("/home/%s/.ssh/%s", vmuser, ssh_rsa_key_file)
	ca_cert_file_path := fmt.Sprintf("/home/%s/%s", vmuser, Ca_cert_file)
	ca_key_file_path := fmt.Sprintf("/home/%s/%s", vmuser, Ca_key_file)
	ray_bootstrap_config_file_path := fmt.Sprintf("/home/%s/%s", vmuser, ray_bootstrap_config_file)
	gen_cert_file_path := fmt.Sprintf("/home/%s/gencert.sh", vmuser)

	// enable tls by default
	cloudConfig.EnableTLS = 1
	// if user sets enable_tls to false in CRD, do not enable tls on Ray's grpc channel
	enable_tls := cloudConfig.VmDeploymentRequest.EnableTLS
	if !enable_tls {
		cloudConfig.EnableTLS = 0
	}

	// Commands to be run on head and worker nodes before ray start
	var setUpCommands []string
	// Create file store map to be produced during cloudinit execution.
	var files []map[string]string
	files = append(files,
		map[string]string{
			"path": ssh_rsa_key_path,

			// Provide padding to the contents of private key file
			// when adding it to cloud init config
			"content": addIndentation(cloudConfig.SshPvtKey, 5),

			// Permission of ssh private key should be 0600
			// ref: https://superuser.com/questions/215504/permissions-on-private-key-in-ssh-folder
			"permissions": "0600",
		},
		map[string]string{
			"path":        ca_cert_file_path,
			"content":     addIndentation(cloudConfig.CaCrt, 5),
			"permissions": "0444",
		},
		map[string]string{
			"path":        ca_key_file_path,
			"content":     addIndentation(cloudConfig.CaKey, 5),
			"permissions": "0444",
		},
		map[string]string{
			"path":        fmt.Sprintf("/home/%s/gencert.sh", vmuser),
			"content":     addIndentation(tls.GetRayTLSConfigString(), 5),
			"permissions": "0777",
		},
	)

	// HeadNodeStatus will be nil if it is the head node.
	var port = RayHeadDefaultPort
	if cloudConfig.VmDeploymentRequest.HeadNodeConfig.Port != nil {
		port = int32(*cloudConfig.VmDeploymentRequest.HeadNodeConfig.Port)
	}

	var docker_cmd string
	var bootstrap_setup_cmd string
	var templ *template.Template
	if cloudConfig.VmDeploymentRequest.HeadNodeStatus == nil {
		bootstrapYamlString, err := convertToYaml(getRayBootstrapConfig(cloudConfig), 5)
		if err != nil {
			return nil, err
		}

		files = append(files,
			map[string]string{
				"path":    fmt.Sprintf("/home/%s/ray_bootstrap_config.yaml", vmuser),
				"content": bootstrapYamlString,
			},
		)

		setup_cmds := append(cloudConfig.VmDeploymentRequest.HeadNodeConfig.SetupCommands,
			fmt.Sprintf(RayHeadStartCmd, port),
		)

		docker_cmd = strings.Join(setup_cmds, ";")

		templ, err = template.New("cloud-config").Parse(cloudConfigHeadNodeTemplate)
		if err != nil {
			return []byte{}, err
		}
	} else {
		var err error
		templ, err = template.New("cloud-config").Parse(cloudConfigWorkerNodeTemplate)
		if err != nil {
			return []byte{}, err
		}
	}
	// commands to run on head node and worker node before ray starts
	setUpCommands = append(setUpCommands,
		RunScriptToGenCerts,
	)
	bootstrap_setup_cmd = strings.Join(setUpCommands, ";")
	buf := bytes.NewBufferString("")
	if err := templ.Execute(buf, map[string]interface{}{
		"users":                          users,
		"files":                          files,
		"docker_image":                   cloudConfig.VmDeploymentRequest.DockerImage,
		"head_node_ip":                   cloudConfig.HeadNodeIp,
		"svc_account_token":              cloudConfig.SvcAccToken,
		"docker_cmd":                     docker_cmd,
		"ray_head_port":                  port,
		"ssh_rsa_key_path":               ssh_rsa_key_path,
		"ray_bootstrap_config_file_path": ray_bootstrap_config_file_path,
		"tls_ca_cert_path":               ca_cert_file_path,
		"tls_ca_key_path":                ca_key_file_path,
		"gen_cert_script_path":           gen_cert_file_path,
		"setup_cmd":                      bootstrap_setup_cmd,
		"enable_tls":                     cloudConfig.EnableTLS,
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

	dataMap := map[string]string{
		CloudInitConfigUserDataKey: base64.StdEncoding.EncodeToString(data),
	}

	// Only set worker private key in head node's secret
	// there is no requirement for us to store it in
	// worker node's secret.
	if len(cloudConfig.HeadNodeIp) == 0 {
		dataMap[SshPrivateKey] = cloudConfig.SshPvtKey
	}

	return &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      cloudConfig.SecretName,
			Namespace: cloudConfig.VmDeploymentRequest.Namespace,
		},
		StringData: dataMap,
	}, nil
}
