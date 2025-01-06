# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import ray
import time
import socket

@ray.remote(num_cpus=1)
def my_task():
  hostname = socket.gethostname()
  print("Collect hostname and sleep for 60 seconds task running in " + hostname)
  time.sleep(10)
  return hostname

def execute_job(ip):
  ray.init(address="ray://" + ip + ":10001")
  return ray.get([my_task.remote() for _ in range (50)])