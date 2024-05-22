# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import os

NAMESPACE_KEY = "NAMESPACE"
VM_IMAGE_KEY = "VM_IMAGE"
KUBE_CONFIG_FILE = "KUBE_CONFIG_FILE"

class OsEnvConfig:
    def __init__(self):
        self.KUBE_CONFIG_FILE = os.getenv(KUBE_CONFIG_FILE, None)
        self.NAMESPACE = os.getenv(NAMESPACE_KEY, None)
        self.VM_IMAGE = os.getenv(VM_IMAGE_KEY, None)
