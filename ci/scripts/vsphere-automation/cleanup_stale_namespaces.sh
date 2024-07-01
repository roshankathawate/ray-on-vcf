#!/usr/bin/env bash
set +x
# Delete all the namespaces which were created by CI pipeline
kubectl get ns -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | { grep ci- || true; } | \
    xargs -I {} --no-run-if-empty python3 ci/scripts/vsphere-automation/client.py -c namespace -o delete -ns "{}"
