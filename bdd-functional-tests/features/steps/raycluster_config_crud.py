# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

from behave import when, then
from kubernetes.client.rest import ApiException

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
        print("Exception when trying to create raycluster `%s` : %s\n" % (name, e))


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
        print("Exception when trying to fetch rayclusterconfig `%s` : %s\n" % (name, e))


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
        print("Exception when trying to create rayclusterconfig `%s` : %s\n" % (name, e))


@then("Validate rayclusterconfig was deleted")
def validate_rayclusterconfig_by_get(context):
    assert context.deleted_rayclusterconfig
