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
	ssh_rsa_key_file           = "id_rsa_ray"
	Ca_cert_file               = "ca.crt"
	Ca_key_file                = "ca.key"
	svc_account_token_env_file = "svc-account-token.env"
	ray_container_name         = "ray_container"
	RayHeadDefaultPort         = int32(6379)
	RayHeadStartCmd            = "ray start --head --port=%d --block --autoscaling-config=/home/ray/ray_bootstrap_config.yaml --dashboard-host=0.0.0.0"
	RayWorkerStartCmd          = "ray start --block --address=%s:%d"
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
{{- if .enable_docker_execution }}
  - su {{ (index .users 0).user }} -c 'apt-get update && apt-get install -y docker'
  - su {{ (index .users 0).user }} -c 'docker pull {{ .docker_image }}'
  - su {{ (index .users 0).user }} -c 'docker run {{ .docker_flags }} {{ .docker_image }}  /bin/bash -c "sudo -i -u root chmod 0777 /home/ray/.ssh/id_rsa_ray; {{ .docker_cmd }}"'
{{- end }}
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
  - su {{ (index .users 0).user }} -c 'ssh-keygen -f {{ .ssh_rsa_key_path }} -t RSA -y > {{ .ssh_rsa_key_path }}.pub'
  - su {{ (index .users 0).user }} -c 'cat {{ .ssh_rsa_key_path }}.pub >> ~/.ssh/authorized_keys'
{{- if .enable_docker_execution }}
  - su {{ (index .users 0).user }} -c 'apt-get update && apt-get install -y docker'
  - su {{ (index .users 0).user }} -c 'docker pull {{ .docker_image }}'
  - su {{ (index .users 0).user }} -c 'docker run {{ .docker_flags }} {{ .docker_image }} /bin/bash -c "{{ .docker_cmd }}"'
{{- end }}
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
	svc_acc_token_env_path := fmt.Sprintf("/home/%s/%s", vmuser, svc_account_token_env_file)

	// Enable tls by default.
	cloudConfig.EnableTLS = 1

	// If user sets enable_tls to false in CRD, do not enable tls on Ray's grpc channel.
	enable_tls := cloudConfig.VmDeploymentRequest.EnableTLS
	if !enable_tls {
		cloudConfig.EnableTLS = 0
	}

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

	// Commands to be run on head and worker nodes before ray start
	var set_up_commands []string = []string{RunScriptToGenCerts}
	var docker_flags []string = []string{
		"--rm",
		fmt.Sprintf("--name %s", ray_container_name),
		"-d",
		"--network host",
		fmt.Sprintf("-v %s:/home/ray/ca.crt", ca_cert_file_path),
		fmt.Sprintf("-v %s:/home/ray/ca.key", ca_key_file_path),
		fmt.Sprintf("-v %s:/home/ray/gencert.sh", gen_cert_file_path),
		fmt.Sprintf("--env \"RAY_USE_TLS=%d\"", cloudConfig.EnableTLS),
		"--env \"RAY_TLS_CA_CERT=/home/ray/ca.crt\"",
		"--env \"RAY_TLS_SERVER_KEY=/home/ray/tls.key\"",
		"--env \"RAY_TLS_SERVER_CERT=/home/ray/tls.crt\"",
	}
	var templ *template.Template
	if cloudConfig.VmDeploymentRequest.HeadNodeStatus == nil {

		var err error
		// Don't create bootstrap config yaml, if requestor is ray cli
		// as it will copy the local file to head node.
		if !cloudConfig.VmDeploymentRequest.RayClusterRequestor.IsRayCli() {
			f, err := getBootstrapYamlContent(cloudConfig)
			if err != nil {
				return nil, err
			}
			files = append(files, f)
		}
		content := fmt.Sprintf("SVC_ACCOUNT_TOKEN=%s", cloudConfig.SvcAccToken)
		files = append(files,
			map[string]string{
				"path":        svc_acc_token_env_path,
				"content":     addIndentation(content, 5),
				"permissions": "0400",
			},
		)

		set_up_commands = append(set_up_commands,
			cloudConfig.VmDeploymentRequest.HeadNodeConfig.SetupCommands...,
		)

		set_up_commands = append(set_up_commands,
			fmt.Sprintf(RayHeadStartCmd, port),
		)

		docker_flags = append(docker_flags,
			fmt.Sprintf("--env-file %s", svc_acc_token_env_path),
			fmt.Sprintf("-v %s:/home/ray/ray_bootstrap_config.yaml", ray_bootstrap_config_file_path),
			fmt.Sprintf("-v %s:/home/ray/.ssh/id_rsa_ray", ssh_rsa_key_path),
		)

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
		set_up_commands = append(set_up_commands, fmt.Sprintf(RayWorkerStartCmd, cloudConfig.HeadNodeIp, port))
	}

	// Docker cmd to be run on head and worker nodes to start ray container.
	var docker_cmd string = strings.Join(set_up_commands, ";")

	buf := bytes.NewBufferString("")
	if err := templ.Execute(buf, map[string]interface{}{
		"users":                   users,
		"files":                   files,
		"docker_image":            cloudConfig.VmDeploymentRequest.DockerImage,
		"docker_cmd":              docker_cmd,
		"docker_flags":            strings.Join(docker_flags, " "),
		"enable_docker_execution": !cloudConfig.VmDeploymentRequest.RayClusterRequestor.IsRayCli(),
		"ssh_rsa_key_path":        ssh_rsa_key_path,
		"svc_acc_token_env_path":  svc_acc_token_env_path,
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

	return &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      cloudConfig.SecretName,
			Namespace: cloudConfig.VmDeploymentRequest.Namespace,
		},
		StringData: dataMap,
	}, nil
}

func getBootstrapYamlContent(cloudConfig CloudConfig) (map[string]string, error) {

	bootstrapYamlString, err := convertToYaml(getRayBootstrapConfig(cloudConfig), 5)
	if err != nil {
		return nil, err
	}

	user := cloudConfig.VmDeploymentRequest.NodeConfig.VMUser
	return map[string]string{
		"path":    fmt.Sprintf("/home/%s/ray_bootstrap_config.yaml", user),
		"content": bootstrapYamlString,
	}, nil
}
