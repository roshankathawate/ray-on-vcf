# RAY Controller Image & Carvel Package Generation

## Ubuntu Prerequisites

Ensure the following tools are installed before setting up the **Taiga project**:

* **Docker** – container runtime
* **Python3** – required for scripts and dependencies
* **virtualenv** – for creating isolated Python environments
* **pip3** – Python package manager
* **Kustomize** – install with:

  ```bash
  1. sudo apt update
  2. curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
  3. sudo mv kustomize /usr/local/bin/
  ```
* **yq (v4.x.x)** – YAML processor, install with:

  ```bash
  sudo apt install yq -y
  ```
* **GNU sed (`gsed`)** – required by the Makefile (create a symlink):

  ```bash
  sudo ln -s $(which sed) /usr/local/bin/gsed
  ```

---

## Generate VMRay Cluster Controller Image

Run the following command to build the controller image:

```bash
make -C vmray-cluster-operator/ vsphere-yaml create-image-tar
```

**Output:**

1. Docker image: `vmray-cluster-controller:latest`
2. TAR archive: `vmray-cluster-operator/artifacts/vmray-cluster-controller.tar.gz`

Next, tag and push the image to a Docker registry accessible from your vCenter:

```bash
docker tag vmray-cluster-controller:latest <DOCKER_REGISTRY>/vmray-cluster-controller:<TAG>
docker push <DOCKER_REGISTRY>/vmray-cluster-controller:<TAG>
```

**Example:**

```bash
docker tag vmray-cluster-controller:latest project-taiga-docker-local.artifactory.vcfd.broadcom.net/vmray-cluster-controller:tag1.0
docker push project-taiga-docker-local.artifactory.vcfd.broadcom.net/vmray-cluster-controller:tag1.0
```

---

## Carvel Package Prerequisites

Set up a Python environment and install required dependencies:

```bash
virtualenv vmray-pyenv
source vmray-pyenv/bin/activate
python3 -m pip install --upgrade pip
pip3 install -r ci/scripts/requirements.txt
```

---

## Export Required Variables

Before generating the Carvel package, export the following environment variables:

```bash
export TAG_NAME=<Tag for Carvel Package>
export CARVEL_PACKAGE_VERSION=<Carvel Package Version>
export TAIGA_SVC_ACCOUNT_USER=<Docker Registry Username>
export TAIGA_SVC_ACCOUNT_PASSWORD=<Docker Registry Password>
export DOCKER_ARTIFACTORY_URL=<Docker Registry URL>
export CARVEL_IMAGE_LOCATION=<Carvel Image Location>
```

**Example:**

```bash
export TAG_NAME=tag1.0
export CARVEL_PACKAGE_VERSION=indian1.0
export TAIGA_SVC_ACCOUNT_USER=user01
export TAIGA_SVC_ACCOUNT_PASSWORD="password12"
export DOCKER_ARTIFACTORY_URL=project-taiga-docker-local.artifactory.vcfd.broadcom.net
export CARVEL_IMAGE_LOCATION=project-taiga-docker-local.artifactory.vcfd.broadcom.net/carvel/taiga
```

---

## Generate Carvel Package

Execute the following scripts to generate the Carvel package:

```bash
python3 ./ci/scripts/vsphere-automation/namespace/update_k8s_params.py -i $DOCKER_ARTIFACTORY_URL/vmray-cluster-controller:$TAG_NAME

./ci/scripts/carvel-packing-local.sh --local

# Carvel package is located at
vmray-cluster-operator/artifacts/carvel-imgpkg/carvel-package-<CARVEL_PACKAGE_VERSION>.yaml
```
