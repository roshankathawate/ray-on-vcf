# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import argparse
import os
from namespace.vcenter_namespace_provider import VcenterNamespaceProvider
from utils import Constants

parser = argparse.ArgumentParser(description='Arguments for vsphere automation sdk')
parser.add_argument('-c', '--category', action='store', help='API Category')
parser.add_argument('-o', '--operation', action='store', help='API Operation - create, list, delete')
parser.add_argument('-ns', '--namespacename', required=False, action='store', help='API Operation - create, list, delete')

args = parser.parse_args()
category = args.category
operation = args.operation
namespace_name = args.namespacename

match category:
    case "namespace":
        client = VcenterNamespaceProvider(os.environ["VCENTER_HOST_NAME"], os.environ["VCENTER_USERNAME"], os.environ["VCENTER_USER_PASSWORD"], Constants.SessionType.UNVERIFIED)
        match operation:
            case "create":
                client.create(os.environ["VCENTER_CLUSTER_NAME"], namespace_name)
            case "list":
                print(client.list())
            case "delete":
                if namespace_name == None:
                    previous_short_commit_id = os.environ["CI_COMMIT_BEFORE_SHA"][:8]
                    instances = client.list()
                    for instance in instances:
                        namespace_name = instance.namespace
                        if previous_short_commit_id in namespace_name:
                            print(f"Deleting namespace {namespace_name}")
                            client.delete(namespace_name)
                else:
                    print(f"Deleting namespace {namespace_name}")
                    client.delete(namespace_name)
