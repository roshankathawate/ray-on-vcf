# Developer Guide

This guide is for developers who want to contribute to RayOnVCF.

## Prerequisites

- Go version v1.20.0+
- Docker version 17.03+
- `make`
- `python3`

It is also recommended to install the pre-commit git hook using the set-up file `.pre-commit-config.yaml`.
This will automatically run go fmt, import, lint, static and copyright check on all committed files.
From the project directory follow these steps to install and enable the pre-commit hook:
```
python3 -m pip install pre-commit
pre-commit install
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.57.2
go install honnef.co/go/tools/cmd/staticcheck@2022.1
```

The tools will be installed in the `$HOME/go/bin` directory by the aforementioned golang install commands, ensure that the `bin` directory is added to the `PATH`.
After installation, the pre-commit hook will trigger on each commit.
If you want to manually run the tools, use `pre-commit run` or `pre-commit run --all-files`.

NOTE: your first run may take a while because environments are being installed and set up. Subsequent runs will be much quicker.

## Python Environment Setup
1. python -m venv ray-env
2. source ray-env/bin/activate
3. pip install -r hack/upload_artifacts/requirements.txt

## Creating and Uploading Artifacts to CPVM
1. GOOS=linux GOARCH=amd64 make -C vmray-cluster-operator/ vsphere-yaml create-image-tar
2. python hack/upload_artifacts/cpvm_upload_artifacts.py -i "CPVM IP" -o -n <NAMESPACE>

## Linting and Formatting:
1. For python, please install black linter for formatting.

## Upstream Gateway Routing for Ingress IP

Setup Env:

1. export WORKING_DIR=hack/setup_dev_proxy
2. python -m venv $WORKING_DIR/ray-env
3. source $WORKING_DIR/ray-env/bin/activate
4. pip install -r $WORKING_DIR/requirements.txt

```
python $WORKING_DIR/setup_gw_proxy.py -s <GW_IP> -u <GW_USER_WITH_ROOT_PRIVILEGE> -ip <UPSTREAM_INGRESS_IP> -pt <UPSTREAM_PORT>
```

Note: By default gw user is root & port is set to ray's default dashboard port i.e. 8265. So above cmd can be reduced to:
```
python $WORKING_DIR/setup_gw_proxy.py -s <GW_IP> -ip <UPSTREAM_INGRESS_IP>
```
