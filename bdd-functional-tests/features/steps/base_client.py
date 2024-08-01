# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

from behave import given
from steps.common.ray_cluster import RayClusterConfig, RayCluster


@given("Setup raycluster config")
def setup_rayclusterconfig(context):
    context.rcc = RayClusterConfig()


@given("Setup vmray cluster k8s client")
def setup_ray_cluster_obj(context):
    if "rcc" not in context:
        context.rcc = None
    context.rc_client = RayCluster(context.rcc)
