# RayOnVCF

This document outlines the user workflow for deploying Ray on VCF.

RayOnVCF caters to three different personas:

- **VI Admin**: Creates a deployment namespace for DevOps users.
- **DevOps User**: Deploys Ray clusters within the created namespace.
- **Data Scientist**: Accesses the Ray Dashboard and submits jobs to the Ray cluster.

## Prerequisites

- Access to a vSphere environment with Workload Control Plane (WCP) enabled.
- `kubectl` version v1.11.3+

## Installing a Ray Cluster on VCF

### VI Admin: Install Supervisor Service and Create Deployment Namespace

1.  **Generate Carvel package**: To generate the carvel package, please refer to the [Generate Carvel Package for RayOnVCF Cluster Operator](#generate-carvel-package-for-RayOnVCF-cluster-operator) section.
2.  **Login to vSphere Client**: Log in to the vSphere client.
3.  **Add New Service**: Navigate to Workload Management -> Services -> Add New Service and upload the carvel.yaml downloaded in step 1.
    If deployment fails, ensure ray-on-vcf.vmware.com is added to the allowed service list.
4.  **Verify Service Status**: Once the service is added, its status should appear as Active in the ray-on-vcf service card.
5.  **Manage Service**: Click on ACTIONS on the ray-on-vcf card -> Manage Service -> Select the supervisor where the service needs to be installed.
6.  **Skip YAML Service Config**: Users do not need to add 'yaml service config' during service installation.
7.  **Verify Service Configuration**: Verify that the ray-on-vcf service status changes to configured
after service installation.
8.  **Verify Service Namespace**: Navigate to Workload Management -> Namespaces tab and verify that a
new service namespace with prefix svc-ray-on-vcf is created on the supervisor cluster where the
service is installed.
9.  **Verify Controller Pod**: Navigate to the service namespace created in step 8 -> compute tab ->
Pods and verify that RayOnVCF-cluster-controller is running inside the pod.
10. **Create Subscribed Content Library**:
    1.  Navigate to Menu -> Content Libraries -> Click Create. The New Content Library wizard opens.
    2.  Specify the Name and location of the content library and click Next.
    3.  Configure the content library subscription at the Configure content library page and click Next.
    4.  Select the Subscribed content library option.
    5.  Enter the Subscription URL address of the publisher: https://packages.vmware.com/dl-vm/lib.json
    6.  For the Download content option, select one of the following:
        *   Immediately: Synchronizes both library metadata and images. Deleted items remain in storage and must be manually deleted.
        *   When needed: Synchronizes only library metadata. Images are downloaded when published. Recommended to save storage space.
    7.  When prompted, accept the SSL certificate thumbprint.
    8.  Configure the OVF security policy at the Apply security policy page and click Next.
    9.  Select Apply Security Policy.
    10. Select OVF default policy. This verifies the OVF signing certificate during synchronization. Templates failing validation are marked
     Verification Failed and their files are not synchronized.
    11. At the Add storage page, select a datastore as a storage location for the content library contents and click Next.
    12. On the Ready to complete page, review the details and click Finish.
11. **Create Deployment Namespace**: Navigate to Workload Management -> Namespaces -> New Namespace
-> create a new namespace on the supervisor cluster with a name, e.g., "deploy-ray".
 cluster with a name, e.g., "deploy-ray".
12. **Attach Resources to Namespace**: Attach VM class, storage policy, and the content library configured in the above steps to this (deploy-ray) namespace.
13. **User Creation (Optional)**: If using the `<administrator@vsphere.local>` user, skip steps 14 and 15. Otherwise, proceed to create a new user.
14. **Create DevOps User**: Navigate to Home -> Administration -> Users and Groups -> Change domain to vsphere.local -> Add username and password.
15. **Add Permissions to DevOps User**: Add permission (edit/view/owner) to the DevOps user created in step 14 to provide required access to the deploy-ray namespace.

### DevOps User: Deploy a Ray Cluster

1.  **Kubectl vSphere Login**: Perform kubectl vsphere login to the supervisor cluster where the ray-on-vcf service is installed with the DevOps user configured previously.
    ```bash
    kubectl vsphere login  --server=<SUPERVISOR Cluster IP> --insecure-skip-tls-verify --vsphere-username <DEVOPS_USER_NAME> --tanzu-kubernetes-cluster-namespace deploy-ray
    ```
2.  **Create RayOnVCFCluster CRD**: Create the RayOnVCFcluster CRD on CPVM. Refer to the samples below for reference.
    ```yaml
    apiVersion: vmray.broadcom.com/v1alpha1
    kind: VMRayCluster
    metadata:
      name: <Provide Any Cluster Name e.g test-ray>
      namespace: <Namespace you created e.g deploy-ray>
    spec:
      api_server:
        location: <Supervisor Cluster IP>
      common_node_config:
        max_workers: 5
        network:
          interfaces:
          - name: eth0
            network:
              apiVersion: vmware.com/v1alpha1
              kind: VirtualNetwork
              name: ""
        available_node_types:
          ray.worker.default:
            max_workers: 5
            min_workers: 2
            resources:
              cpu: 1
            vm_class: <VM Class you want to use e.g. best-effort-xlarge>
          ray.head.default:
            vm_class: <VM Class you want to use e.g. best-effort-xlarge>
        storage_class: <Storage attached to the namespace e.g.vsan-default-storage-policy>
        vm_image: <any VMI associated with the namespace e.g. vmi-7289e7d90a3c53e73. Refer step 5 to get a VMI>
        vm_password_salt_hash: $6$test1234$9/BUZHNkvq.c1miDDMG5cHLmM4V7gbYdGuF0//3gSIh//DOyi7ypPCs6EAA9b8/tidHottL6UG0tG/RqTgAAi/
        vm_user: ray
      head_node:
        port: 6245
        node_type: ray.head.default
      ray_docker_image: <Docker image you want to use e.g. rayproject/ray:latest>
    ```
3.  **Update Ray Docker Image**: Update the `ray_docker_image`
    *   For CPU only: `rayproject/ray:latest`
    *   For GPU support: `rayproject/ray:latest-gpu`
4.  **Find VM Image (VMI)**: Find the VM image (VMI) associated with the deployment namespace using the command:
    ```bash
    kubectl -n deploy-ray get vmi
    ```
5.  **Apply CRDs**: Apply the CRDs using the command:
    ```bash
    kubectl apply -f <crd.yaml>
    ```
6.  **Verify Ray Cluster Status**: Verify that the Ray cluster comes up under the deployment namespace.
    (This will take some time as VMs need to be provisioned).
7.  **Monitor Ray Cluster**: Monitor the status of the Raycluster using `kubectl get`.
     For more detailed information, use the `-o yaml` switch.

### Generating the Carvel Package
These steps describe how to generate and upload a carvel package using the `ci/scripts/carvel-packing.sh` script.
1.  **Build artifacts**: Create operator artifacts are by running:
    ```bash
    make -C vmray-cluster-operator/ vsphere-yaml create-image-tar
    ```
2.  **Setup Python environment**:
    Create a python virtual environment and install dependencies.
    ```bash
    python3 -m venv carvel-env
    source carvel-env/bin/activate
    pip install -r ci/scripts/requirements.txt
    ```
3.  **Set environment variables**: The packaging script requires several environment variables to be set.
    ```bash
    export DOCKER_ARTIFACTORY_URL=<your_docker_artifactory_url>
    export TAG_NAME=<your_image_tag> # e.g., 1.0.123-abcdef
    export CARVEL_PACKAGE_VERSION=<your_package_version> # e.g, 1.0.123
    export TAIGA_SVC_ACCOUNT_USER=<artifactory_user>
    export TAIGA_SVC_ACCOUNT_PASSWORD=<artifactory_password>
    export TAIGA_GENERIC_REPOSITORY_URL=<artifactory_generic_repo_url>
    export PACKAGE_TYPE=<development_or_production> # "development" or "production"
    ```
4.  **Update Kubernetes manifests**: Update the image reference in the Kubernetes manifest files.
    This step is necessary before creating the carvel package.
    ```bash
    python3 ci/scripts/vsphere-automation/namespace/update_k8s_params.py -i "$DOCKER_ARTIFACTORY_URL/vmray-cluster-controller:$TAG_NAME"
    ```
5.  **Run packaging script**:
    ```bash
    ./ci/scripts/carvel-packing.sh
    ```
This will generate the carvel package and upload it to Artifactory.