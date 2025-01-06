# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

from behave import when, then
from kubernetes.client.rest import ApiException
from steps.utils import (
    get_desired_worker_count,
    get_minimum_worker_count,
    match_worker_nodes_status,
    match_node_status,
    fetch_vm_service_ip,
)
from steps.base_client import set_ft_value, get_ft_value

import json
import logging
import os
import time
import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

@when("Submit rayclusterconfig named `{name}`")
def submit_rayclusterconfig_obj(context, name):
    namespace = context.rc_client.os_env_config.NAMESPACE
    assert namespace
    try:
        context.rc_client.CreateRayCluster(namespace, name)
        set_ft_value(context, name, "create-vmraycluster", True)
    except ApiException as e:
        logging.info(
            "Exception when trying to create raycluster `%s` : %s\n" % (name, e)
        )
        set_ft_value(context, name, "create-vmraycluster", False)


@when("Fetch ray clusterconfig `{name}` from WCP env")
def validate_rayclusterconfig_exists(context, name):
    try:
        namespace = context.rc_client.os_env_config.NAMESPACE
        assert namespace

        resp = context.rc_client.GetRayCluster(namespace, name)
        correct_cr = resp["metadata"]["name"] == name

        set_ft_value(context, name, "fetch-vmraycluster", correct_cr)
    except ApiException as e:
        logging.info(
            "Exception when trying to fetch rayclusterconfig `%s` : %s\n" % (name, e)
        )
        set_ft_value(context, name, "fetch-vmraycluster", False)


@when("Delete rayclusterconfig named `{name}`")
def delete_rayclusterconfig(context, name):
    try:
        namespace = context.rc_client.os_env_config.NAMESPACE
        assert namespace

        context.rc_client.DeleteRayCluster(namespace, name)
        set_ft_value(context, name, "delete-vmraycluster", True)
    except ApiException as e:
        logging.info(
            "Exception when trying to create rayclusterconfig `%s` : %s\n" % (name, e)
        )
        set_ft_value(context, name, "delete-vmraycluster", False)

@when("Delete `{number:d}` worker nodes of the ray-cluster named `{name}`")
def delete_rayworkernodes(context, name, number):
    try:
        namespace = context.rc_client.os_env_config.NAMESPACE
        assert namespace
        context.deleted_vms_count = 0
        
        deleted_req_vms = False
        vms = context.rc_client.ListNodes(namespace)
        for vm in vms.get('items', []):
            vm_name = vm['metadata']['name']
            if vm_name.startswith(name + "-w-"):
                context.rc_client.DeleteVM(namespace, vm_name)
                context.deleted_vms_count = context.deleted_vms_count + 1
                logging.info(
                        "Deleted worker nodes `%s`" % (vm_name)
                )
            if number == context.deleted_vms_count:
                deleted_req_vms = True
                logging.info("Deleted required number of worker nodes")
                break
        set_ft_value(context, name, "delete-workers", deleted_req_vms)
    except ApiException as e:
        logging.info(
            "Exception when trying to trying to delete worker nodes `%s` : %s\n" % (name, e)
        )
        set_ft_value(context, name, "delete-workers", False)


@when(
    "Check in interval of `{interval_time:d}` seconds if head node for raycluster `{name}` is up in `{wait_time:d}` seconds"
)
def wait_for_head_node_to_come_up(context, interval_time, name, wait_time):
    try:
        head_is_up = False

        namespace = context.rc_client.os_env_config.NAMESPACE
        assert namespace

        end_time = time.time() + wait_time
        while time.time() < end_time:

            resp = context.rc_client.GetRayCluster(namespace, name)
            if "head_node_status" in resp.get("status", {}):
                if match_node_status(resp["status"]["head_node_status"], "running"):
                    head_is_up = True
                    break

            logging.info(
                "Head node is not ready, waiting for %d seconds\n" % (interval_time)
            )
            time.sleep(interval_time)
        set_ft_value(context, name, "head-is-up", head_is_up)
    except ApiException as e:
        logging.info(
            "Exception when trying to check if head node is up for raycluster `%s` : %s\n"
            % (name, e)
        )


@when(
    "Check in interval of `{interval_time:d}` seconds if worker nodes for raycluster `{name}` are up in `{wait_time:d}` seconds"
)
def wait_for_worker_nodes_to_come_up(context, interval_time, name, wait_time):
    is_head_up = get_ft_value(context, name, "head-is-up")
    if is_head_up == False or is_head_up is None:
        logging.info("Skipping, because head node is not up")
        return

    # Wait until workers come up.
    try:

        end_time = time.time() + wait_time
        workers_are_up = False
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
                    workers_are_up = True
                    break

            logging.info(
                "Worker nodes are not ready, waiting for %d seconds\n" % (interval_time)
            )
            time.sleep(interval_time)
        set_ft_value(context, name, "workers-are-up", workers_are_up)
    except ApiException as e:
        logging.info(
            "Exception when trying to check if worker nodes are up for raycluster `%s` : %s\n"
            % (name, e)
        )
        set_ft_value(context, name, "workers-are-up", False)

@when("Submit a job named `{job}` to ray cluster `{name}`")
def submit_job(context, job, name):

    namespace = context.rc_client.os_env_config.NAMESPACE
    assert namespace

    are_workers_up = get_ft_value(context, name, "workers-are-up")
    if are_workers_up == False or are_workers_up is None:
        logging.info("Skipping, because worker nodes are not up")
        return

    ip = fetch_vm_service_ip(context.rc_client, name)
    logging.info("Fetched vm service ip for submitting job: " + ip)
    job_executed = False

    # Get minimum numbers of worker nodes that are active.
    resp = context.rc_client.GetRayCluster(namespace, name)
    min_active_wrks = get_minimum_worker_count(resp["spec"])
    total_wrks = min_active_wrks + 1 # include headnode
    logging.info("Total workers: " + str(total_wrks))

    # Read tls crt and key secret, and set them for ray tls
    tls_secret_name = name + "-tls"
    root_cert_secret_name = name + "-root-cert"
    context.rc_client.ReadSecretTofile(namespace, tls_secret_name, "tls.crt", "/tmp/ray-cluster-tls.crt", logging)
    context.rc_client.ReadSecretTofile(namespace, tls_secret_name, "tls.key", "/tmp/ray-cluster-tls.key", logging)
    context.rc_client.ReadSecretTofile(namespace, root_cert_secret_name, "ca.cert", "/tmp/ray-cluster-ca.crt", logging)

    os.environ["RAY_USE_TLS"]         = "1"
    os.environ["RAY_TLS_CA_CERT"]     = "/tmp/ray-cluster-ca.crt"
    os.environ["RAY_TLS_SERVER_CERT"] = "/tmp/ray-cluster-tls.crt"
    os.environ["RAY_TLS_SERVER_KEY"]  = "/tmp/ray-cluster-tls.key"

    # TODO: wait until ray image is pulled and started in all worker.
    # [REMOVE] currently we don't have ability to figure out ray
    # cluster status so using sleep cmd
    time.sleep(300)

    if job == "sleep":
        from steps.jobs.sleep import execute_job
        out = set(execute_job(ip))
        logging.info("Outcome of sleep job: " + str(out))
        resp = context.rc_client.GetRayCluster(namespace, name)
        desired_worker_count = get_desired_worker_count(resp["spec"])
        logging.info("Desired worker count, scaling up is requested: {}".format(desired_worker_count))
        job_executed = len(out) >= total_wrks and desired_worker_count > total_wrks
    set_ft_value(context, name, "job-" + job + "-executed", job_executed)

@then("Validate operations performed on raycluster `{name}`")
def validate_ops(context, name):
    vdict = context.validation_container[name]
    for k, v in vdict.items():
        logging.info("Validating outcome for operation `%s`: %s"% (k, v))
        assert v


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

