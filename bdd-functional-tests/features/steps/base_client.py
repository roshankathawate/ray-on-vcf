# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

from behave import given
from steps.common.ray_cluster import NodeConfig, RayClusterConfig, RayCluster
from steps.common.os_env_config import OsEnvConfig


@given("Setup node config")
def setup_nodeconfig(context):
    oec = OsEnvConfig()
    assert oec.VM_IMAGE
    context.nc = NodeConfig(oec.VM_IMAGE)


@given("Setup raycluster config")
def setup_rayclusterconfig(context):
    context.rcc = RayClusterConfig()


@given("Setup vmray cluster k8s client")
def setup_ray_cluster_obj(context):
    if "rcc" not in context:
        context.rcc = None
    if "nc" not in context:
        context.nc = None
    context.rc_client = RayCluster(context.rcc, context.nc)
