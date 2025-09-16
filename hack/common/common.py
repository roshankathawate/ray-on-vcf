# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import paramiko
from scp import SCPClient
from getpass import getpass

def create_ssh_scp_client(ip, user):
   # Create ssh client.
   ssh_client = paramiko.SSHClient()
   ssh_client.load_system_host_keys()
   ssh_client.connect(hostname=ip,
               username=user,
               password=getpass())

   # Create scp client.
   scp = SCPClient(ssh_client.get_transport())
   return ssh_client, scp