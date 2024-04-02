# pip3 install paramiko

import os
import argparse
from getpass import getpass
import paramiko
import shutil
import ruamel.yaml
import sys
from scp import SCPClient
import tarfile

parser = argparse.ArgumentParser(description="Upload vmray cluster operator docker image to resgistry")
parser.add_argument('-i', dest='ip', type=str, required=True)
parser.add_argument('-u', dest='user', default="root")
parser.add_argument(
        "-o",
        dest="deploy_operator",
        required=False,
        action="store_true",
        help="True if you would also like to deploy the operator on the CPVM",
    )
parser.add_argument(
   "-n", dest="namespace", default="vmw-system-vmrayclusterop", required=False
)

args = parser.parse_args()

# Create ssh client.
ssh_client = paramiko.SSHClient()
ssh_client.load_system_host_keys()
ssh_client.connect(hostname=args.ip, 
            username=args.user,
            password=getpass())

# Create scp client.
scp = SCPClient(ssh_client.get_transport())

source_artifacts = ["crd.yaml", "vmray-cluster-controller.tar.gz", "vsphere-deployment-manager.yaml", "vsphere-deployment-rbac.yaml"]
upload_artifacts = ["crd_2.yaml", "vmray-cluster-controller.tar.gz", "vsphere-deployment-manager_2.yaml", "vsphere-deployment-rbac_2.yaml"]

# Find relative path to artifacts
dirname = os.getcwd()
artifacts_path = os.path.join(dirname, 'vmray-cluster-operator/artifacts/')

for i, artifact in enumerate(source_artifacts):
   source_artifact_path = os.path.join(artifacts_path, artifact)
   
   if not os.path.isfile(source_artifact_path):
      print(f"File {source_artifact_path} does not exist")
      sys.exit(1)
      
   # Make a copy of the existing files
   try:
      shutil.copyfile(source_artifact_path, os.path.join(artifacts_path, upload_artifacts[i]))
   except shutil.SameFileError as e:
      print(f"Skipping the file {artifact} as the upload artifact is same as source")
      
if args.namespace:
   for i, artifact in enumerate(upload_artifacts):
      current_artifact_path = os.path.join(artifacts_path, artifact)
      
      # If its a tarfile then we don't process it
      if tarfile.is_tarfile(current_artifact_path):
         continue
   
      # Update namespace in all the YAML files
      with open(current_artifact_path, 'r') as file:
         filedata = file.read()

      # Replace namespace with new one.
      filedata = filedata.replace("namespace: vmw-system-vmrayclusterop", f"namespace: {args.namespace}")

      # Write the file out again
      with open(current_artifact_path, 'w') as file:
         file.write(filedata)
      
if args.deploy_operator:
   vsphere_deployment_manager_bkup_path = os.path.join(artifacts_path, "vsphere-deployment-manager_2.yaml")
   # Update the image field
   config, ind, bsi = ruamel.yaml.util.load_yaml_guess_indent(open(vsphere_deployment_manager_bkup_path))
   config["spec"]["template"]["spec"]["containers"][0]["image"] = f"{args.ip}:5000/vmware/vmray-cluster-controller:latest"
   yaml = ruamel.yaml.YAML()
   yaml.indent(mapping=ind, sequence=ind, offset=bsi)

   with open(vsphere_deployment_manager_bkup_path, 'w') as fp:
      yaml.dump(config, fp)
   
remote_dir = "/tmp/vmray-cluster-op-artifacts/"

ssh_client.exec_command('rm -rf ' + remote_dir)
ssh_client.exec_command('mkdir -p ' + remote_dir)

# List artifacts to upload.
for fn in upload_artifacts:
   print(f"upload {os.path.join(artifacts_path,fn)} to {remote_dir+fn}")
   scp.put(os.path.join(artifacts_path,fn), remote_dir+fn)

scp.close()

# Leverage skopeo cmd to upload image in tar.gz to repo via ssh.
tar_file = remote_dir + "vmray-cluster-controller.tar.gz"
imagepath = "vmware/vmray-cluster-controller:latest"

skopep_cmd = "skopeo --insecure-policy copy --dest-tls-verify=false  docker-archive:%s docker://localhost:5002/%s"%(tar_file, imagepath)

print(f"Executing {skopep_cmd}")

if args.deploy_operator:
   _, stdout, stderr = ssh_client.exec_command(skopep_cmd)
   exit_status = stdout.channel.recv_exit_status()
   if exit_status == 0:
      print("Skopeo command executed successfully")
   else:
      print(f"Skopeo command failed {stderr}")
      sys.exit(1)
   
   _, stdout, stderr = ssh_client.exec_command(f"kubectl get namespace {args.namespace}")
   exit_status = stdout.channel.recv_exit_status()
   if exit_status != 0:
      print("Namespace does not exists. Please create it from Workload Management on the vSphere")
      sys.exit(1)
        
   for i, artifact in enumerate(upload_artifacts):
      # If its a tarfile then we don't process it
      if tarfile.is_tarfile(os.path.join(artifacts_path, source_artifacts[i])):
         continue
      
      artifact_remote_path = os.path.join(remote_dir, artifact)
      
      _, stdout, stderr = ssh_client.exec_command(f"kubectl apply -f {artifact_remote_path}")
      exit_status = stdout.channel.recv_exit_status()
      if exit_status == 0:
         print(f"Successful applied {artifact}")
      else:
         print(f"Error appliying {artifact}: {stderr}")
         sys.exit(1)
      
   sys.exit(0)

print("Execute following cmd in CPVM to upload image to bootstrap registry:\n", skopep_cmd)