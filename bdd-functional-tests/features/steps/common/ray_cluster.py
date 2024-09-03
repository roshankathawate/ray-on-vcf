# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import kubernetes
from .os_env_config import OsEnvConfig


class RayClusterConfig:
    def __init__(self, name: str = "ray-cluster", min_workers: int = 3):
        self.name = name
        self.min_workers = min_workers
        self.os_env_config = OsEnvConfig()

        # hash of `raydebian` using salt `test1234`
        self.vm_password_salt_hash = "$6$test1234$9/BUZHNkvq.c1miDDMG5cHLmM4V7gbYdGuF0//3gSIh//DOyi7ypPCs6EAA9b8/tidHottL6UG0tG/RqTgAAi/"
        self.vm_user = "ray-vm"

    def get_cluster_cr(self, namespace: str, name: str):
        clusterconfig = {}
        clusterconfig["apiVersion"] = "vmray.broadcom.com/v1alpha1"
        clusterconfig["kind"] = "VMRayCluster"

        metadata = {}
        metadata["namespace"] = namespace
        metadata["name"] = name
        clusterconfig["metadata"] = metadata

        spec = {}

        api_server = {}
        api_server["ca_cert"] = ""
        api_server["location"] = self.os_env_config.K8S_SERVER_IP
        spec["api_server"] = api_server

        head_node = {}
        head_node["setup_commands"] = []
        head_node["port"] = 6254
        head_node["node_type"] = "head.node"
        spec["head_node"] = head_node

        common_node_config = {}
        common_node_config["storage_class"] = self.os_env_config.STORAGE_CLASS
        common_node_config["vm_image"] = self.os_env_config.VM_IMAGE
        common_node_config["vm_password_salt_hash"] = self.vm_password_salt_hash
        common_node_config["vm_user"] = self.vm_user
        common_node_config["max_workers"] = 5

        available_node_types = {}
        available_node_types["head.node"] = {
            "vm_class": self.os_env_config.VM_CLASS,
            "max_workers": 1,
            "min_workers": 0,
            "resources": {
                "cpu": 0,
            },
        }
        available_node_types["ray.worker.default"] = {
            "vm_class": self.os_env_config.VM_CLASS,
            "max_workers": 5,
            "min_workers": 2,
            "resources": {
                "cpu": 4,
                "memory": 4294967296,  # 4096 MBs in bytes.
            },
        }

        common_node_config["available_node_types"] = available_node_types

        spec["common_node_config"] = common_node_config
        spec["ray_docker_image"] = (
            "project-taiga-docker-local.artifactory.eng.vmware.com/development/ray:milestone_2.1"
        )
        clusterconfig["spec"] = spec

        return clusterconfig


class RayCluster:

    def __init__(self, rayconfig: RayClusterConfig):

        self.vmray_group = "vmray.broadcom.com"
        self.vmray_version = "v1alpha1"
        self.vmray_cluster_cr_plural = "vmrayclusters"
        self.vmop_group = "vmoperator.vmware.com"
        self.vmop_version = "v1alpha3"

        self.os_env_config = OsEnvConfig()

        kubeconfigfile = self.os_env_config.KUBE_CONFIG_FILE
        kubernetes.config.load_kube_config(config_file=kubeconfigfile)

        self.custom_resource_client = kubernetes.client.CustomObjectsApi()
        self.rayconfig = rayconfig

    def CreateRayCluster(self, namespace: str, name: str):
        # Apply ray cluster config.
        rc = self.rayconfig.get_cluster_cr(namespace=namespace, name=name)
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
    
    def ListVMs(self, namespace: str):
        return self.custom_resource_client.list_namespaced_custom_object(
             group=self.vmop_group,
             version=self.vmop_version,
             namespace=namespace,
             plural="virtualmachines",
    )

    def DeleteVM(self, namespace: str, vm_name: str):
        return self.custom_resource_client.delete_namespaced_custom_object(
                            group=self.vmop_group,
                            version=self.vmop_version,
                            namespace=namespace,
                            plural="virtualmachines",
                            name=vm_name
                        )
