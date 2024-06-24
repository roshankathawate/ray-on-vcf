# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import os
import requests
import urllib3
from vmware.vapi.vsphere.client import create_vsphere_client
from com.vmware.vcenter.namespaces_client import Instances, Access
from utils import get_unverified_session, get_configuration, Constants

class VcenterNamespaceProvider:
    def __init__(self, server, user, password, session_type: Constants.SessionType):
        self.server = server
        self.user = user
        self.password = password
        self.session_type = session_type
        self.vcenter_namespace_client = self.get_client()

    def get_client(self):
        session = None
        if self.session_type == Constants.SessionType.UNVERIFIED:
            session = get_unverified_session()
        # TODO: support verified context
        stub_config = get_configuration(
                self.server, self.user, self.password,
                session)
        return Instances(stub_config)

    def create(self, cluster_name, namespace_name, description):
        storage_spec = [Instances.StorageSpec(policy=os.environ["STORAGE_POLICY"])]
        vmservice_spec = Instances.VMServiceSpec(vm_classes={os.environ["VM_CLASS"]}, content_libraries={os.environ["CONTENT_LIBRARY_UUID"]})
        access_list = [Instances.Access(subject_type=Access.SubjectType.USER, subject=os.environ["DEVOPS_USERNAME"], domain=os.environ["USER_DOMAIN_NAME"], role=Access.Role.EDIT)]
        instance_spec = Instances.CreateSpec(cluster=cluster_name, namespace=namespace_name, description=description, storage_specs=storage_spec, vm_service_spec=vmservice_spec, access_list=access_list)
        self.vcenter_namespace_client.create(instance_spec)

    def delete(self, namespace_name):
        self.vcenter_namespace_client.delete(namespace_name)

    def list(self):
        return self.vcenter_namespace_client.list()

    def get(self, namespace_name):
        try:
            self.vcenter_namespace_client.get(namespace_name)
        except Exception as exc:
            print(f"Error encountered in get(), Error {exc}")
