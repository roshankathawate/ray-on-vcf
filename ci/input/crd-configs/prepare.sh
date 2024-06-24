#!/usr/bin/env bash
set -x

# Update namespace and supervisor ip
sed -i "s/<SUPERVISOR_NAMESPACE>/$NAMESPACE_NAME-deploy-ray/g" ci/input/crd-configs/vmraycluster.yaml
sed -i "s/<CPVM_IP>/$SUPERVISOR_IP/g" ci/input/crd-configs/vmraycluster.yaml

# Update namespace, storageclass and vmi
sed -i "s/<SUPERVISOR_NAMESPACE>/$NAMESPACE_NAME-deploy-ray/g" ci/input/crd-configs/vmraynodeconfig.yaml
sed -i "s/<VMI_IMAGE>/$VMI/g" ci/input/crd-configs/vmraynodeconfig.yaml
sed -i "s/<STORAGE_CLASS>/$STORAGE_CLASS/g" ci/input/crd-configs/vmraynodeconfig.yaml
