# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import kubernetes
import os
from .os_env_config import OsEnvConfig


class RayClusterConfig:
    def __init__(self, name: str = "ray-cluster", min_workers: int = 3):
        self.name = name
        self.min_workers = min_workers
        self.os_env_config = OsEnvConfig()

    def get_cluster_cr(self, namespace: str, name: str, nodeconfig_name: str):
        clusterconfig = {}
        clusterconfig["apiVersion"] = "vmray.broadcom.com/v1alpha1"
        clusterconfig["kind"] = "VMRayCluster"

        metadata = {}
        metadata["namespace"] = namespace
        metadata["name"] = name
        clusterconfig["metadata"] = metadata

        spec = {}
        spec["desired_workers"] = []

        api_server = {}
        api_server["ca_cert"] = ""
        api_server["CPVM_IP"] = self.os_env_config.K8S_SERVER_IP
        spec["api_server"] = api_server

        head_node = {}
        head_node["setup_commands"] = []
        head_node["port"] = 6254
        head_node["node_config_name"] = nodeconfig_name
        spec["head_node"] = head_node

        worker_node = {}
        worker_node["max_workers"] = 5
        worker_node["min_workers"] = 3
        worker_node["node_config_name"] = nodeconfig_name
        worker_node["idle_timeout_minutes"] = 5
        spec["worker_node"] = worker_node

        spec["ray_docker_image"] = "project-taiga-docker-local.artifactory.eng.vmware.com/development/ray:milestone_1"

        clusterconfig["spec"] = spec

        return clusterconfig


class NodeConfig:
    def __init__(self, vm_image: str):
        self.os_env_config = OsEnvConfig()
        self.vm_image = vm_image

        # hash of `raydebian` using salt `test1234`
        self.vm_password_salt_hash = "$6$test1234$9/BUZHNkvq.c1miDDMG5cHLmM4V7gbYdGuF0//3gSIh//DOyi7ypPCs6EAA9b8/tidHottL6UG0tG/RqTgAAi/"
        self.vm_user = "ray-vm"

    def get_nodeconfig_cr(self, namespace: str, name: str):
        nodeconfig = {}
        nodeconfig["apiVersion"] = "vmray.broadcom.com/v1alpha1"
        nodeconfig["kind"] = "VMRayNodeConfig"

        metadata = {}
        metadata["namespace"] = namespace
        metadata["name"] = name
        nodeconfig["metadata"] = metadata

        spec = {}
        spec["storage_class"] = self.os_env_config.STORAGE_CLASS
        spec["vm_class"] = self.os_env_config.VM_CLASS
        spec["vm_image"] = self.os_env_config.VM_IMAGE

        spec["vm_password_salt_hash"] = self.vm_password_salt_hash
        spec["vm_user"] = self.vm_user
        nodeconfig["spec"] = spec

        return nodeconfig


class RayCluster:

    def __init__(self, rayconfig: RayClusterConfig, nodeconfig: NodeConfig):

        self.vmray_group = "vmray.broadcom.com"
        self.vmray_version = "v1alpha1"
        self.vmray_cluster_cr_plural = "vmrayclusters"
        self.vmray_nodeconfig_cr_plural = "vmraynodeconfigs"
        self.os_env_config = OsEnvConfig()

        kubeconfigfile = self.os_env_config.KUBE_CONFIG_FILE
        kubernetes.config.load_kube_config(config_file=kubeconfigfile)

        self.custom_resource_client = kubernetes.client.CustomObjectsApi()
        self.rayconfig = rayconfig
        self.nodeconfig = nodeconfig

    def CreateNodeConfig(self, namespace: str, name: str):
        # Apply node config.
        nc = self.nodeconfig.get_nodeconfig_cr(namespace=namespace, name=name)
        return self.custom_resource_client.create_namespaced_custom_object(
            self.vmray_group,
            self.vmray_version,
            namespace,
            self.vmray_nodeconfig_cr_plural,
            nc,
        )

    def GetNodeConfig(self, namespace: str, name: str):
        return self.custom_resource_client.get_namespaced_custom_object(
            self.vmray_group,
            self.vmray_version,
            namespace,
            self.vmray_nodeconfig_cr_plural,
            name,
        )

    def DeleteNodeConfig(self, namespace: str, name: str):
        return self.custom_resource_client.delete_namespaced_custom_object(
            self.vmray_group,
            self.vmray_version,
            namespace,
            self.vmray_nodeconfig_cr_plural,
            name,
        )

    def CreateRayCluster(self, namespace: str, name: str, nodeconfig_name):
        # Apply ray cluster config.
        rc = self.rayconfig.get_cluster_cr(namespace=namespace, name=name, nodeconfig_name=nodeconfig_name)
        return self.custom_resource_client.create_namespaced_custom_object(
            self.vmray_group,
            self.vmray_version,
            namespace,
            self.vmray_cluster_cr_plural,
            rc,
        )

    def GetRayCluster(self, namespace: str, name: str):
        return self.custom_resource_client.get_namespaced_custom_object(
            self.vmray_group,
            self.vmray_version,
            namespace,
            self.vmray_cluster_cr_plural,
            name,
        )

    def DeleteRayCluster(self, namespace: str, name: str):
        return self.custom_resource_client.delete_namespaced_custom_object(
            self.vmray_group,
            self.vmray_version,
            namespace,
            self.vmray_cluster_cr_plural,
            name,
        )
