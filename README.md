# VMRay


### Setup python env:
1. python -m venv ray-env
2. source ray-env/bin/activate
3. pip install -r hack/requirements.txt

### Create CRDs, deployment yamls, image artifacts to be uploaded to CPVM (i.e. Supervisior Control Plane VM):
1. GOOS=linux GOARCH=amd64 make -C vmray-cluster-operator/ kustomize-crd kustomize-vsphere-deploy-yaml create-image-tar
2. python hack/cpvm_upload_artifacts.py -i "CPVM IP"

Apply CRDs, roles, role bindings & operator deployment:
1. ssh into CPVM
2. Apply crds : `kubectl apply -f /tmp/vmray-cluster-op-artifacts/crd.yaml`
3. Using vsphere APIs create a user namespace called `vmw-system-vmrayclusterop` to deploy operator in.
4. Apply roles & rolebindings : `kubectl apply -f /tmp/vmray-cluster-op-artifacts/vsphere-deployment-rbac.yaml`
5. Update image attribute in `/tmp/vmray-cluster-op-artifacts/vsphere-deployment-manager.yaml` to `<CPVM IP>:5000/vmware/vmray-cluster-controller:latest`
5. Create operator deployment : `kubectl apply -f /tmp/vmray-cluster-op-artifacts/vsphere-deployment-manager.yaml`