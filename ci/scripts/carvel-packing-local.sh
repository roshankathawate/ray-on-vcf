#!/usr/bin/env bash
set -e

echo "🔍 Parsing arguments..."
# Parse arguments
LOCAL_MODE=false
for arg in "$@"; do
  case $arg in
    --local)
      LOCAL_MODE=true
      shift
      ;;
    *)
      ;;
  esac
done
echo "   LOCAL_MODE=${LOCAL_MODE}"

echo "📂 Changing directory to vmray-cluster-operator/artifacts"
cd vmray-cluster-operator/artifacts

echo "📁 Creating directory structure carvel-imgpkg/taiga"
mkdir -p carvel-imgpkg/taiga

echo "📄 Copying ytt-vsphere.yaml → carvel-imgpkg/taiga/config.yml"
cp ytt-vsphere.yaml carvel-imgpkg/taiga/config.yml

echo "📝 Creating ytt values.yml file"
echo -e "#@data/values\n---\napp_name: taiga" > carvel-imgpkg/taiga/values.yml

echo "📂 Changing directory to carvel-imgpkg"
cd carvel-imgpkg

echo "📁 Creating .imgpkg directory"
mkdir -p .imgpkg

echo "⚙️  Generating images.yml with kbld"
kbld -f taiga/config.yml --imgpkg-lock-output .imgpkg/images.yml

echo "🔑 Logging into Docker registry: ${DOCKER_ARTIFACTORY_URL}"
echo "${TAIGA_SVC_ACCOUNT_PASSWORD}" | docker login "${DOCKER_ARTIFACTORY_URL}" -u "${TAIGA_SVC_ACCOUNT_USER}" --password-stdin

echo "📦 Pushing image bundle to ${CARVEL_IMAGE_LOCATION}:${CARVEL_PACKAGE_VERSION}"
imgpkg push -b ${CARVEL_IMAGE_LOCATION}:${CARVEL_PACKAGE_VERSION} -f ../carvel-imgpkg

echo "📝 Creating carvel-package-${CARVEL_PACKAGE_VERSION}.yaml file"
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
  template:
    spec:
      fetch:
      - imgpkgBundle:
          image: ${CARVEL_IMAGE_LOCATION}:<TAG_NAME>
      template:
        - ytt:
            paths:
              - taiga
      deploy:
        - kapp: {}
EOF

echo "🔧 Replacing <TAG_NAME> with ${CARVEL_PACKAGE_VERSION}"
sed -i "s/<TAG_NAME>/${CARVEL_PACKAGE_VERSION}/g" carvel-package-${CARVEL_PACKAGE_VERSION}.yaml

if [ "$LOCAL_MODE" = false ]; then
  echo "⬆️  Uploading carvel-package-${CARVEL_PACKAGE_VERSION}.yaml to ${TAIGA_GENERIC_REPOSITORY_URL}/carvel-package-yaml/${PACKAGE_TYPE}/"
  curl -u "${TAIGA_SVC_ACCOUNT_USER}":"${TAIGA_SVC_ACCOUNT_PASSWORD}" \
       -T carvel-package-${CARVEL_PACKAGE_VERSION}.yaml \
       "${TAIGA_GENERIC_REPOSITORY_URL}/carvel-package-yaml/${PACKAGE_TYPE}/"
  echo "✅ Upload complete"
else
  echo "⏭️  Skipping upload since --local option was provided."
fi
