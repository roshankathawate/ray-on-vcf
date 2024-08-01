# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

from steps.common.constants import DEFAULT_HEAD_TYPE


def match_node_status(node_status, desired_status):
    if node_status.get("vm_status", None) == desired_status:
        return True
    return False


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
            if k != DEFAULT_HEAD_TYPE:
                count = count + v.get("min_workers", 0)
    return count
