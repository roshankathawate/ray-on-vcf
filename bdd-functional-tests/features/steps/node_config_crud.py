# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

from behave import when, then
from kubernetes.client.rest import ApiException

import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)


@when("Submit nodeconfig named `{name}`")
def submit_nodeconfig_obj(context, name):
    namespace = context.rc_client.os_env_config.NAMESPACE
    assert namespace
    try:
        context.rc_client.CreateNodeConfig(namespace, name)
        context.nodeconfig_created = True
    except ApiException as e:
        print("Exception when trying to create nodeconfig `%s` : %s\n" % (name, e))


@then("Validate vmraynodeconfig was created")
def validate_nodeconfig_creation(context):
    assert context.nodeconfig_created


@when("Fetch ray nodeconfig `{name}` from WCP env")
def validate_nodeconfig_exists(context, name):
    try:
        namespace = context.rc_client.os_env_config.NAMESPACE
        assert namespace

        resp = context.rc_client.GetNodeConfig(namespace, name)
        context.fetched_nodeconfig = resp["metadata"]["name"] == name
    except ApiException as e:
        print("Exception when trying to fetch nodeconfig `%s` : %s\n" % (name, e))


@then("Validate correct vmraynodeconfig was fetched")
def validate_nodeconfig_by_get(context):
    assert context.fetched_nodeconfig


@when("Delete nodeconfig named `{name}`")
def delete_nodeconfig(context, name):
    try:
        namespace = context.rc_client.os_env_config.NAMESPACE
        assert namespace

        context.rc_client.DeleteNodeConfig(namespace, name)
        context.deleted_nodeconfig = True
    except ApiException as e:
        print("Exception when trying to create nodeconfig `%s` : %s\n" % (name, e))


@then("Validate vmraynodeconfig was deleted")
def validate_nodeconfig_by_get(context):
    assert context.deleted_nodeconfig
