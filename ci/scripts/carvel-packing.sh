#!/usr/bin/env bash
set -x

# Update the namespace to ytt template value
python3 ci/scripts/vsphere-automation/namespace/update_k8s_params.py -cn $NAMESPACE_NAME -n "#@ data.values.namespace"

cd vmray-cluster-operator/artifacts
mkdir -p carvel-imgpkg/taiga
cat <<EOF > carvel-imgpkg/taiga/config.yml
#@ load("@ytt:data", "data")

#@ def labels():
app: "ray-on-vcf"
#@ end
EOF

# copy all kubernetes manifest files content to config.yml
for f in *.yaml
do
  echo "---" >> carvel-imgpkg/taiga/config.yml
  cat $f >> carvel-imgpkg/taiga/config.yml
done

# Create a ytt template file
echo -e "#@data/values\n---\napp_name: taiga" > carvel-imgpkg/taiga/values.yml

# Create .imgpkg directory
cd carvel-imgpkg
mkdir -p .imgpkg

# Generate images.yml file
kbld -f taiga/config.yml --imgpkg-lock-output .imgpkg/images.yml

# Push image to taiga docker repository
imgpkg push -b project-taiga-docker-local.artifactory.eng.vmware.com/carvel/taiga:$CARVEL_PACKAGE_VERSION -f ../carvel-imgpkg

# Create carvel-package.yaml file
cat <<EOF > carvel-package-$CARVEL_PACKAGE_VERSION.yaml
---
apiVersion: data.packaging.carvel.dev/v1alpha1
kind: PackageMetadata
metadata:
  name: ray-on-vcf.vmware.com
spec:
  displayName: ray-on-vcf
  shortDescription: ray-on-vcf service
---
apiVersion: data.packaging.carvel.dev/v1alpha1
kind: Package
metadata:
  name: ray-on-vcf.vmware.com.<TAG_NAME>
spec:
  refName: ray-on-vcf.vmware.com
  version: <TAG_NAME>
  template: # type of App CR
    spec:
      fetch:
      # An imgpkg bundle is an OCI image that contains Kubernetes configurations.
      # Refer to carvel-imgpkg/README for steps of building a bundle.
      - imgpkgBundle:
          image: project-taiga-docker-local.artifactory.eng.vmware.com/carvel/taiga:<TAG_NAME>
      template:
        - ytt:
            paths:
              - taiga
      deploy:
        - kapp: {}
EOF

# Replace the image version in the carvel-package.yaml
sed -i "s/<TAG_NAME>/$CARVEL_PACKAGE_VERSION/g" carvel-package-$CARVEL_PACKAGE_VERSION.yaml

# Upload carvel-package.yaml to artifactory
curl -u "$TAIGA_SVC_ACCOUNT_USER":"$TAIGA_SVC_ACCOUNT_PASSWORD" -T carvel-package-$CARVEL_PACKAGE_VERSION.yaml "${TAIGA_GENERIC_REPOSITORY_URL}/carvel-package-yaml/"
