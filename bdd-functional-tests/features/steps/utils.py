# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

from kubernetes.client.rest import ApiException


def match_node_status(node_status, desired_status):
    return  node_status.get("vm_status", None) == desired_status


def match_worker_nodes_status(current_workers, worker_names, desired_status):
    for w in worker_names:
        if w not in current_workers or not match_node_status(
            current_workers[w], desired_status
        ):
            return False
    return True


def get_minimum_worker_count(spec):
    count = 0
    if spec is not None and "available_node_types" in spec.get(
        "common_node_config", {}
    ):
        available_node_types = spec["common_node_config"]["available_node_types"]
        for k, v in available_node_types.items():
            count = count + v.get("min_workers", 0)
    return count

def get_desired_worker_count(spec):
    if spec is None:
        return 0
    return len(spec.get("autoscaler_desired_workers", {}).keys())

def fetch_vm_service_ip(rc_client, name):
    try:
        namespace = rc_client.os_env_config.NAMESPACE
        assert namespace

        resp = rc_client.GetRayCluster(namespace, name)
        return resp["status"]["vm_service_status"]["ip"]
    except ApiException as e:
        logging.info(
            "Exception when trying to fetch rayclusterconfig `%s` : %s\n" % (name, e)
        )
    return None
