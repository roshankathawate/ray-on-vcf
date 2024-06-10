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

DEFAULT_NAMESPACE = "vmware-system-rayclusterop"

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
   "-n", dest="namespace", default=DEFAULT_NAMESPACE, required=False
)

args = parser.parse_args()

def create_ssh_scp_client(ip, user):
   # Create ssh client.
   ssh_client = paramiko.SSHClient()
   ssh_client.load_system_host_keys()
   ssh_client.connect(hostname=args.ip,
               username=args.user,
               password=getpass())

   # Create scp client.
   scp = SCPClient(ssh_client.get_transport())
   return ssh_client, scp

def relative_path():
   # Find relative path to artifacts
   dirname = os.getcwd()
   return os.path.join(dirname, 'vmray-cluster-operator/artifacts/')

def copy_artifacts(source_artifacts, upload_artifacts):
   artifacts_path = relative_path()
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

def update_namespace(namespace, upload_artifacts):
   artifacts_path = relative_path()
   for i, artifact in enumerate(upload_artifacts):
      current_artifact_path = os.path.join(artifacts_path, artifact)

      # If its a tarfile then we don't process it
      if tarfile.is_tarfile(current_artifact_path):
         continue

      # Update namespace in all the YAML files
      with open(current_artifact_path, 'r') as file:
         filedata = file.read()

      # Replace namespace with new one.
      filedata = filedata.replace(DEFAULT_NAMESPACE, f"{namespace}")

      # Write the file out again
      with open(current_artifact_path, 'w') as file:
         file.write(filedata)

def update_operator_img(ip, port, image_path):
   artifacts_path = relative_path()
   vsphere_deployment_manager_bkup_path = os.path.join(artifacts_path, "vsphere-deployment-manager_2.yaml")
   # Update the image field
   config, ind, bsi = ruamel.yaml.util.load_yaml_guess_indent(open(vsphere_deployment_manager_bkup_path))
   config["spec"]["template"]["spec"]["containers"][0]["image"] = f"{ip}:{port}/{image_path}"
   yaml = ruamel.yaml.YAML()
   yaml.indent(mapping=ind, sequence=ind, offset=bsi)

   with open(vsphere_deployment_manager_bkup_path, 'w') as fp:
      yaml.dump(config, fp)

def list_upload_artifacts(scp, remote_dir, upload_artifacts):
   # List artifacts to upload.
   for fn in upload_artifacts:
      artifacts_path = relative_path()
      print(f"upload {os.path.join(artifacts_path,fn)} to {remote_dir+fn}")
      scp.put(os.path.join(artifacts_path,fn), remote_dir+fn)

def execute_skopep_cmd(skopep_cmd, namespace, source_artifacts, upload_artifacts):
   print(f"Executing {skopep_cmd}")
   artifacts_path = relative_path()
   _, stdout, stderr = ssh_client.exec_command(skopep_cmd)
   exit_status = stdout.channel.recv_exit_status()
   if exit_status == 0:
      print("Skopeo command executed successfully")
   else:
      print(f"Skopeo command failed {stderr}")
      sys.exit(1)

   _, stdout, stderr = ssh_client.exec_command(f"kubectl get namespace {namespace}")
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

##
# Execute script.
##

src_artifacts = ["crd.yaml", "vmray-cluster-controller.tar.gz",  "vsphere-deployment-webhook.yaml", "vsphere-deployment-manager.yaml", "vsphere-deployment-rbac.yaml",]
upld_artifacts = ["crd_2.yaml", "vmray-cluster-controller.tar.gz", "vsphere-deployment-webhook_2.yaml", "vsphere-deployment-manager_2.yaml", "vsphere-deployment-rbac_2.yaml"]

ssh_client, scp = create_ssh_scp_client(args.ip, args.user)
copy_artifacts(src_artifacts, upld_artifacts)

if args.namespace:
   update_namespace(args.namespace, upld_artifacts)

remote_dir = "/tmp/vmray-cluster-op-artifacts/"
image_path = "vmware/vmray-cluster-controller:latest"
image_port = 5000

ssh_client.exec_command('rm -rf ' + remote_dir)
ssh_client.exec_command('mkdir -p ' + remote_dir)

if args.deploy_operator:
   update_operator_img(args.ip, image_port, image_path)

list_upload_artifacts(scp, remote_dir, upld_artifacts)
scp.close()

# Leverage skopeo cmd to upload image in tar.gz to repo via ssh.
tar_file = remote_dir + "vmray-cluster-controller.tar.gz"
skopep_cmd = "skopeo --insecure-policy copy --dest-tls-verify=false  docker-archive:%s docker://localhost:5002/%s"%(tar_file, image_path)

if args.deploy_operator:
   execute_skopep_cmd(skopep_cmd, args.namespace, src_artifacts, upld_artifacts)

print("Execute following cmd in CPVM to upload image to bootstrap registry:\n", skopep_cmd)
