# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

from behave import when, then
from kubernetes.client.rest import ApiException
from steps.utils import (
    get_minimum_worker_count,
    match_worker_nodes_status,
    match_node_status,
)

import json
import logging
import time
import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)


@when("Submit rayclusterconfig named `{name}`")
def submit_rayclusterconfig_obj(context, name):
    namespace = context.rc_client.os_env_config.NAMESPACE
    assert namespace
    try:
        context.rc_client.CreateRayCluster(namespace, name)
        context.rayclusterconfig_created = True
    except ApiException as e:
        logging.info(
            "Exception when trying to create raycluster `%s` : %s\n" % (name, e)
        )


@then("Validate rayclusterconfig was created")
def validate_rayclusterconfig_creation(context):
    assert context.rayclusterconfig_created


@when("Fetch ray clusterconfig `{name}` from WCP env")
def validate_rayclusterconfig_exists(context, name):
    try:
        namespace = context.rc_client.os_env_config.NAMESPACE
        assert namespace

        resp = context.rc_client.GetRayCluster(namespace, name)
        context.fetched_rayclusterconfig = resp["metadata"]["name"] == name
    except ApiException as e:
        logging.info(
            "Exception when trying to fetch rayclusterconfig `%s` : %s\n" % (name, e)
        )


@then("Validate correct rayclusterconfig was fetched")
def validate_rayclusterconfig_by_get(context):
    assert context.fetched_rayclusterconfig


@when("Delete rayclusterconfig named `{name}`")
def delete_rayclusterconfig(context, name):
    try:
        namespace = context.rc_client.os_env_config.NAMESPACE
        assert namespace

        context.rc_client.DeleteRayCluster(namespace, name)
        context.deleted_rayclusterconfig = True
    except ApiException as e:
        logging.info(
            "Exception when trying to create rayclusterconfig `%s` : %s\n" % (name, e)
        )


@then("Validate rayclusterconfig was deleted")
def validate_rayclusterconfig_by_get(context):
    assert context.deleted_rayclusterconfig


@when(
    "Check in interval of `{interval_time:d}` seconds if head node for raycluster `{name}` is up in `{wait_time:d}` seconds"
)
def wait_for_head_node_to_come_up(context, interval_time, name, wait_time):
    try:
        if "cluster_head_nodes" not in context:
            context.cluster_head_nodes = {}

        context.cluster_head_nodes[name] = False

        end_time = time.time() + wait_time

        while time.time() < end_time:
            namespace = context.rc_client.os_env_config.NAMESPACE
            assert namespace

            resp = context.rc_client.GetRayCluster(namespace, name)
            if "head_node_status" in resp.get("status", {}):
                if match_node_status(resp["status"]["head_node_status"], "running"):
                    context.cluster_head_nodes[name] = True
                    break

            logging.info(
                "Head node is not ready, waiting for %d seconds\n" % (interval_time)
            )
            time.sleep(interval_time)
    except ApiException as e:
        logging.info(
            "Exception when trying to check if head node is up for raycluster `%s` : %s\n"
            % (name, e)
        )


@then("Validate that head node came up for raycluster `{name}`")
def validate_if_head_node_is_up(context, name):
    assert context.cluster_head_nodes.get(name, False)


@when(
    "Check in interval of `{interval_time:d}` seconds if worker nodes for raycluster `{name}` are up in `{wait_time:d}` seconds"
)
def wait_for_worker_nodes_to_come_up(context, interval_time, name, wait_time):
    try:
        if "cluster_worker_nodes_up" not in context:
            context.cluster_worker_nodes_up = {}

        context.cluster_worker_nodes_up[name] = False

        end_time = time.time() + wait_time

        while time.time() < end_time:
            namespace = context.rc_client.os_env_config.NAMESPACE
            assert namespace

            resp = context.rc_client.GetRayCluster(namespace, name)

            # Get minimum numbers of worker nodes requested.
            worker_node_count = get_minimum_worker_count(resp["spec"])

            # Check if all the worker nodes are in running state.
            autoscaler_desired_workers = resp["spec"].get(
                "autoscaler_desired_workers", {}
            )
            desired_worker_nodes_names = autoscaler_desired_workers.keys()

            if worker_node_count <= len(desired_worker_nodes_names):
                match_wn = match_worker_nodes_status(
                    resp["status"].get("current_workers", {}),
                    desired_worker_nodes_names,
                    "running",
                )
                if match_wn:
                    context.cluster_worker_nodes_up[name] = True
                    break

            logging.info(
                "Worker nodes are not ready, waiting for %d seconds\n" % (interval_time)
            )
            time.sleep(interval_time)
    except ApiException as e:
        logging.info(
            "Exception when trying to check if worker nodes are up for raycluster `%s` : %s\n"
            % (name, e)
        )


@then("Validate that desired worker nodes came up for raycluster `{name}`")
def validate_if_head_node_is_up(context, name):
    assert context.cluster_worker_nodes_up.get(name, False)


@then("Log status of raycluster named `{name}` in case of failure")
def print_raycluster(context, name):
    try:
        namespace = context.rc_client.os_env_config.NAMESPACE
        assert namespace

        resp = context.rc_client.GetRayCluster(namespace, name)
        logging.info("Raycluster CR:" + json.dumps(resp, indent=2))
    except ApiException as e:
        logging.info(
            "Exception when trying to fetch rayclusterconfig `%s` : %s\n" % (name, e)
        )
