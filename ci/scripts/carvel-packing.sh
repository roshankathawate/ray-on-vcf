#!/usr/bin/env bash
set -e

cd vmray-cluster-operator/artifacts
mkdir -p carvel-imgpkg/taiga
cp ytt-vsphere.yaml carvel-imgpkg/taiga/config.yml

# Create a ytt template file
echo -e "#@data/values\n---\napp_name: taiga" > carvel-imgpkg/taiga/values.yml

# Create .imgpkg directory
cd carvel-imgpkg
mkdir -p .imgpkg

# Generate images.yml file
kbld -f taiga/config.yml --imgpkg-lock-output .imgpkg/images.yml

# Push image to taiga docker repository
docker login -u "${TAIGA_SVC_ACCOUNT_USER}" -p "${TAIGA_SVC_ACCOUNT_PASSWORD}" "${DOCKER_ARTIFACTORY_URL}"
imgpkg push -b project-taiga-docker-local.artifactory.vcfd.broadcom.net/carvel/taiga:${CARVEL_PACKAGE_VERSION} -f ../carvel-imgpkg

# Create carvel-package.yaml file
cat <<EOF > carvel-package-${CARVEL_PACKAGE_VERSION}.yaml
---
apiVersion: data.packaging.carvel.dev/v1alpha1
kind: PackageMetadata
metadata:
  name: ray-on-vcf.broadcom.com
spec:
  displayName: ray-on-vcf
  shortDescription: ray-on-vcf service
---
apiVersion: data.packaging.carvel.dev/v1alpha1
kind: Package
metadata:
  name: ray-on-vcf.broadcom.com.<TAG_NAME>
spec:
  refName: ray-on-vcf.broadcom.com
  version: <TAG_NAME>
  template: # type of App CR
    spec:
      fetch:
      # An imgpkg bundle is an OCI image that contains Kubernetes configurations.
      # Refer to carvel-imgpkg/README for steps of building a bundle.
      - imgpkgBundle:
          image: project-taiga-docker-local.artifactory.vcfd.broadcom.net/carvel/taiga:<TAG_NAME>
      template:
        - ytt:
            paths:
              - taiga
      deploy:
        - kapp: {}
EOF

# Replace the image version in the carvel-package.yaml
sed -i "s/<TAG_NAME>/${CARVEL_PACKAGE_VERSION}/g" carvel-package-${CARVEL_PACKAGE_VERSION}.yaml

# Upload carvel-package.yaml to artifactory
curl -u "${TAIGA_SVC_ACCOUNT_USER}":"${TAIGA_SVC_ACCOUNT_PASSWORD}" -T carvel-package-${CARVEL_PACKAGE_VERSION}.yaml "${TAIGA_GENERIC_REPOSITORY_URL}/carvel-package-yaml/${PACKAGE_TYPE}/"