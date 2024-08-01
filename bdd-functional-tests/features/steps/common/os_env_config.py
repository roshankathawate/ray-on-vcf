# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import os

NAMESPACE_KEY = "NAMESPACE"
KUBE_CONFIG_FILE = "KUBE_CONFIG_FILE"


class OsEnvConfig:
    def __init__(self):
        self.KUBE_CONFIG_FILE = os.getenv(KUBE_CONFIG_FILE, None)
        self.NAMESPACE = os.getenv(NAMESPACE_KEY, None)
        self.VM_IMAGE = os.getenv("VMI", None)
        self.K8S_SERVER_IP = os.getenv("SUPERVISOR_IP", None)
        self.VM_CLASS = os.getenv("VM_CLASS", None)
        self.STORAGE_CLASS = os.getenv("STORAGE_CLASS", None)
