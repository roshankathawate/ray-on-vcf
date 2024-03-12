# pip3 install paramiko

import os
import argparse
from getpass import getpass
import paramiko

from scp import SCPClient

parser = argparse.ArgumentParser(description="Upload vmray cluster operator docker image to resgistry")
parser.add_argument('-i', dest='ip', type=str, required=True)
parser.add_argument('-u', dest='user', default="root")
args = parser.parse_args()

# Create ssh client.
ssh_client = paramiko.SSHClient()
ssh_client.load_system_host_keys()
ssh_client.connect(hostname=args.ip, 
            username=args.user,
            password=getpass())

# Create scp client.
scp = SCPClient(ssh_client.get_transport())

# Find relative path to artifacts
dirname = os.path.dirname(__file__)
artifacts_path = os.path.join(dirname, '../vmray-cluster-operator/artifacts/')

remote_dir = "/tmp/vmray-cluster-op-artifacts/"

ssh_client.exec_command('rm -rf ' + remote_dir)
ssh_client.exec_command('mkdir -p ' + remote_dir)

# List artifacts to upload.
upload_artifacts = ["crd.yaml", "vray-cluster-controller.tar.gz", "vsphere-deployment-manager.yaml", "vsphere-deployment-rbac.yaml"]
for fn in upload_artifacts:
   scp.put(os.path.join(artifacts_path,fn), remote_dir+fn)

# Leverage skopeo cmd to upload image in tar.gz to repo via ssh.
tarfile = remote_dir + "vray-cluster-controller.tar.gz"
imagepath = "vmware/vray-cluster-controller:latest"

skopep_cmd = "skopeo --insecure-policy copy --dest-tls-verify=false  docker-archive:%s docker://localhost:5002/%s"%(tarfile, imagepath)

print("Execute following cmd in CPVM to upload image to bootstrap registry:\n", skopep_cmd)

scp.close()



