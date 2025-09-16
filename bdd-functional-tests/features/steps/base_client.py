# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

from behave import given
import logging
from steps.common.ray_cluster import RayClusterConfig, RayCluster


@given("Setup raycluster config")
def setup_rayclusterconfig(context):
    context.rcc = RayClusterConfig()


@given("Setup vmray cluster k8s client")
def setup_ray_cluster_obj(context):
    if "rcc" not in context:
        context.rcc = None
    context.rc_client = RayCluster(context.rcc)

@given("Create validation container for cluster `{name}`")
def create_validator(context, name):
    if "validation_container" not in context:
        context.validation_container = {}
    context.validation_container[name] = {}


def set_ft_value(context, name, key, value):
    vdict = context.validation_container[name]
    if key not in vdict:
        context.validation_container[name][key] = value
        logging.info("Setting %s to %s"%(name, value))
    else:
        context.validation_container[name][key] = value and vdict[key]
        logging.info("Setting %s to %s"%(name, value and vdict[key]))

def get_ft_value(context, name, key):
    vdict = context.validation_container[name]
    if key in vdict:
        logging.info("Found %s, where value is %s"%(name, context.validation_container[name][key]))
        return context.validation_container[name][key]
    return None