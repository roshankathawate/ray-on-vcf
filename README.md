# VMRay


### Setup python env:
1. python -m venv ray-env
2. source ray-env/bin/activate
3. pip install -r hack/upload_artifacts/requirements.txt

### Create CRDs, deployment yamls, image artifacts to be uploaded to CPVM (i.e. Supervisior Control Plane VM):
1. GOOS=linux GOARCH=amd64 make -C vmray-cluster-operator/ kustomize-crd kustomize-vsphere-deploy-yaml create-image-tar
2. python hack/upload_artifacts/cpvm_upload_artifacts.py -i "CPVM IP" -o -n <NAMESPACE>

Apply CRDs, roles, role bindings & operator deployment:
1. ssh into CPVM
2. Apply crds : `kubectl apply -f /tmp/vmray-cluster-op-artifacts/crd.yaml`
3. Using vsphere APIs create a user namespace called `vmw-system-vmrayclusterop` to deploy operator in.
4. Apply roles & rolebindings : `kubectl apply -f /tmp/vmray-cluster-op-artifacts/vsphere-deployment-rbac.yaml`
5. Update image attribute in `/tmp/vmray-cluster-op-artifacts/vsphere-deployment-manager.yaml` to `<CPVM IP>:5000/vmware/vmray-cluster-controller:latest`
5. Create operator deployment : `kubectl apply -f /tmp/vmray-cluster-op-artifacts/vsphere-deployment-manager.yaml`

### Steps to upload ovf to content library from local dir
1. Activate python env: source ray-env/bin/activate
2. Get needed reqs: pip install -r hack/content_library/requirements.txt
3. Create a local dir where ovf, vmdk etc are stored to be upload as content item
4. Execute cmd:
   ```
   python hack/content_library/cl-upload.py -i '<VC_IP>' \
      -u 'administrator@vsphere.local'\
      -ds '{datastore-id}' \
      -ln '{library-name}' \
      -lin '{library-item-name}' \
      -lid '{library-item-description}'
      -p {path to ovf dir}
   ```

## Development guids
1. For python, please install [black](https://pypi.org/project/black/) linter for formatting.

## Pre-commit hooks
It is recommended to install the pre-commit git hook using the set-up file `.pre-commit-config.yaml`.
This will automatically run go fmt, import, lint, static and copyright check on all committed files.
From the project directory follow these steps to install and enable the pre-commit hook:
```
python3 -m pip install pre-commit
pre-commit install
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.57.2
go install honnef.co/go/tools/cmd/staticcheck@2022.1
```

The tools will be installed in the `$HOME/go/bin` directory by the aforementioned golang install commands, ensure that the `bin` directory is added to the `PATH`
After installation, the pre-commit hook will trigger on each commit.
If you want to manually run the tools, use `pre-commit run` or `pre-commit run --all-files`

NOTE: your first run may take a while because environments are being installed and set up. Subsequent runs will be much quicker.
