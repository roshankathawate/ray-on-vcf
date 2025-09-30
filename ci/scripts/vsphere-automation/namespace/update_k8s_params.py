# Copyright (c) 2024 VMware by Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

import os
import argparse
from getpass import getpass
import tarfile

parser = argparse.ArgumentParser(description="Update Kubernetes Parameters")
parser.add_argument(
   "-n", dest="namespace", default="vmware-system-rayclusterop", required=False
)

parser.add_argument(
   "-cn", dest="current_namespace", default="vmware-system-rayclusterop", required=False
)

parser.add_argument(
   "-i", dest="image", default="vmray-cluster-controller:latest", required=False
)
args = parser.parse_args()

# Find relative path to artifacts
dirname = os.getcwd()
artifacts_path = os.path.join(dirname, 'vmray-cluster-operator/artifacts/')

source_artifacts = ["crd.yaml", "vmray-cluster-controller.tar.gz", "vsphere-deployment-manager.yaml", "vsphere-deployment-rbac.yaml", "vsphere-deployment-webhook.yaml", "ytt-vsphere.yaml"]

for i, artifact in enumerate(source_artifacts):
    current_artifact_path = os.path.join(artifacts_path, artifact)
    file_updated = False

    # If its a tarfile then we don't process it
    if tarfile.is_tarfile(current_artifact_path):
        continue

    # Update namespace in all the YAML files
    with open(current_artifact_path, 'r') as file:
        filedata = file.read()

    # Replace namespace with new one.
    if args.namespace:
        filedata = filedata.replace(f"namespace: {args.current_namespace}", f"namespace: {args.namespace}")
        file_updated = True
    # Replace image with new one.
    if args.image and (artifact == "vsphere-deployment-manager.yaml" or artifact == "ytt-vsphere.yaml"):
        filedata = filedata.replace("image: vmray-cluster-controller:latest", f"image: {args.image}")
        file_updated = True
    if args.namespace and (artifact == "vsphere-deployment-webhook.yaml"):
        filedata = filedata.replace(args.current_namespace, args.namespace)
        file_updated = True
    # Write the file out again
    if file_updated:
        with open(current_artifact_path, 'w') as file:
            file.write(filedata)
