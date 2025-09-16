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
        self._check_env_vars()
        
    def _check_env_vars(self):
        missing_vars = []
        vars_dict = {
            'KUBE_CONFIG_FILE': self.KUBE_CONFIG_FILE,
            'NAMESPACE': self.NAMESPACE,
            'VM_IMAGE': self.VM_IMAGE,
            'K8S_SERVER_IP': self.K8S_SERVER_IP,
            'VM_CLASS': self.VM_CLASS,
            'STORAGE_CLASS': self.STORAGE_CLASS
        }
        for var_name,value in vars_dict.items():
            if value is None:
                missing_vars.append(var_name)
        
        if missing_vars:
            raise EnvironmentError(f"Missing required environment variables: {', '.join(missing_vars)}")
