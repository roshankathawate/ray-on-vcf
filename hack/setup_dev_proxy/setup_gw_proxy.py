# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import sys
import os

# make common directory importable.
dir_path = os.path.dirname(os.path.realpath(__file__)) + "/../common"
sys.path.append(dir_path)

import argparse
from common import create_ssh_scp_client

if __name__ == "__main__":

   parser = argparse.ArgumentParser(description="Setup tiny proxy on gateway with upstream mapping")
   parser.add_argument('-s', dest='server', type=str, required=True)
   parser.add_argument('-u', dest='user', default="root")
   parser.add_argument('-ip', dest='upstream_ip', type=str, required=True)
   parser.add_argument('-pt', dest='port', type=int, default=8265)

   args = parser.parse_args()

   # step 1: Create ssh client.
   ssh, _ = create_ssh_scp_client(args.server, args.user)

   # step 2: Run cmds to install tiny proxy, append upstream rule
   # into /etc/tinyproxy.conf and finally restart the service.
   ssh.exec_command("apt-get install tinyproxy -y")
   ssh.exec_command("echo \"## Dev setup to route to ingress IP in wcp + nsx setup: Start\n\" >> /etc/tinyproxy.conf")
   ssh.exec_command("echo \"upstream {}:{}\n\" >> /etc/tinyproxy.conf".format(args.upstream_ip, args.port))
   ssh.exec_command("echo \"## End\n\" >> /etc/tinyproxy.conf")
   ssh.exec_command("service tinyproxy restart")

   print("Proxy is setup now.")


